package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type GeminiExtractor struct {
	apiKey      string
	model       string
	endpoint    string
	httpClient  *http.Client
	maxAttempts int
	baseBackoff time.Duration
	maxBackoff  time.Duration
	sleep       func(context.Context, time.Duration) error
	logger      *slog.Logger
}

type GeminiExtractorConfig struct {
	APIKey      string
	Model       string
	Endpoint    string
	HTTPClient  *http.Client
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	Sleep       func(context.Context, time.Duration) error
	Logger      *slog.Logger
}

const (
	defaultMaxAttempts = 3
	defaultBaseBackoff = 1 * time.Second
	defaultMaxBackoff  = 8 * time.Second
)

func NewGeminiExtractor(cfg GeminiExtractorConfig) (*GeminiExtractor, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		panic("gemini api key is required")
	}

	if cfg.Model == "" {
		panic("gemini model is required")
	}

	if cfg.Endpoint == "" {
		panic("gemini endpoint is required")
	}

	model := strings.TrimSpace(cfg.Model)
	apiKey := strings.TrimSpace(cfg.APIKey)
	endpoint := strings.TrimSpace(cfg.Endpoint)

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	baseBackoff := cfg.BaseBackoff
	if baseBackoff <= 0 {
		baseBackoff = defaultBaseBackoff
	}

	maxBackoff := cfg.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = defaultMaxBackoff
	}

	sleep := cfg.Sleep
	if sleep == nil {
		sleep = sleepWithContext
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &GeminiExtractor{
		apiKey:      apiKey,
		model:       model,
		endpoint:    strings.TrimRight(endpoint, "/"),
		httpClient:  httpClient,
		maxAttempts: maxAttempts,
		baseBackoff: baseBackoff,
		maxBackoff:  maxBackoff,
		sleep:       sleep,
		logger:      logger,
	}, nil
}

func (e *GeminiExtractor) ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error) {
	if len(pdfData) == 0 {
		return nil, fmt.Errorf("pdf data is required")
	}

	body, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{
				{Text: prompt},
				{InlineData: &geminiInlineData{MimeType: "application/pdf", Data: base64.StdEncoding.EncodeToString(pdfData)}},
			},
		}},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0,
			ResponseMimeType: "application/json",
			ResponseSchema:   geminiResponseSchema,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", e.endpoint, e.model, e.apiKey)

	var respData []byte
	var statusCode int

	for attempt := 1; attempt <= e.maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build gemini request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, reqErr := e.httpClient.Do(req)
		if reqErr != nil {
			if !isRetryableRequestError(reqErr) || attempt == e.maxAttempts {
				return nil, fmt.Errorf("gemini request failed: %w", reqErr)
			}

			delay := e.backoffDuration(attempt)
			e.logger.Warn("retrying gemini request due to network error", "attempt", attempt, "max_attempts", e.maxAttempts, "delay", delay, "error", reqErr.Error())

			if err := e.sleep(ctx, delay); err != nil {
				return nil, fmt.Errorf("gemini request failed: %w", reqErr)
			}
			continue
		}

		respData, err = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read gemini response: %w", err)
		}

		statusCode = resp.StatusCode
		if statusCode >= 200 && statusCode < 300 {
			break
		}

		if !isRetryableResponse(statusCode, respData) || attempt == e.maxAttempts {
			return nil, fmt.Errorf("gemini response status=%d body=%s", statusCode, string(respData))
		}

		delay := retryDelayFromResponse(resp.Header.Get("Retry-After"), respData)
		if delay <= 0 {
			delay = e.backoffDuration(attempt)
		}

		e.logger.Warn("retrying gemini request due to response status", "attempt", attempt, "max_attempts", e.maxAttempts, "delay", delay, "status", statusCode)

		if err := e.sleep(ctx, delay); err != nil {
			return nil, fmt.Errorf("gemini response status=%d body=%s", statusCode, string(respData))
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("gemini response status=%d body=%s", statusCode, string(respData))
	}

	var payload geminiResponse
	if err := json.Unmarshal(respData, &payload); err != nil {
		return nil, fmt.Errorf("decode gemini response: %w", err)
	}

	text := firstCandidateText(payload)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("gemini response did not include structured output")
	}

	structured := stripCodeFence(text)
	invoice, err := decodeStrictInvoice(structured)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(invoice.CUFE) == "" {
		return nil, domain.ErrMissingCUFE
	}
	if strings.TrimSpace(invoice.Issuer.Name) == "" || strings.TrimSpace(invoice.Issuer.TaxID) == "" {
		return nil, domain.ErrMissingIssuer
	}
	if strings.TrimSpace(invoice.Receiver.Name) == "" || strings.TrimSpace(invoice.Receiver.TaxID) == "" {
		return nil, domain.ErrMissingReceiver
	}
	if len(invoice.Lines) == 0 {
		return nil, domain.ErrMissingLineItems
	}

	invoice.RawData = []byte(structured)
	return invoice, nil
}

func (e *GeminiExtractor) backoffDuration(attempt int) time.Duration {
	if attempt <= 1 {
		return e.baseBackoff
	}

	delay := e.baseBackoff << (attempt - 1)
	if delay > e.maxBackoff {
		return e.maxBackoff
	}

	return delay
}

func isRetryableRequestError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

func isRetryableResponse(statusCode int, body []byte) bool {
	if statusCode >= 500 {
		return true
	}

	if statusCode != http.StatusTooManyRequests {
		return false
	}

	bodyText := strings.ToLower(string(body))
	if strings.Contains(bodyText, "limit: 0") {
		return false
	}

	return true
}

func retryDelayFromResponse(retryAfterHeader string, body []byte) time.Duration {
	if delay := parseRetryAfterHeader(retryAfterHeader); delay > 0 {
		return delay
	}

	return parseRetryDelayFromBody(body)
}

func parseRetryAfterHeader(header string) time.Duration {
	retryAfter := strings.TrimSpace(header)
	if retryAfter == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if retryAt, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(retryAt)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

func parseRetryDelayFromBody(body []byte) time.Duration {
	var payload struct {
		Error struct {
			Details []struct {
				Type       string `json:"@type"`
				RetryDelay string `json:"retryDelay"`
			} `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}

	for _, detail := range payload.Error.Details {
		if detail.Type != "type.googleapis.com/google.rpc.RetryInfo" {
			continue
		}

		retryDelay := strings.TrimSpace(detail.RetryDelay)
		if retryDelay == "" {
			continue
		}

		delay, err := time.ParseDuration(retryDelay)
		if err != nil || delay <= 0 {
			continue
		}

		return delay
	}

	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func decodeStrictInvoice(raw string) (*domain.InvoiceDocument, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()

	var out llmInvoiceOutput
	if err := decoder.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode structured invoice output: %w", err)
	}

	taxTotals := make([]domain.TaxTotal, 0, len(out.TaxTotals))
	for _, t := range out.TaxTotals {
		taxTotals = append(taxTotals, domain.TaxTotal{
			TaxAmount: t.TaxAmount,
			Taxable:   t.Taxable,
			TaxCode:   t.TaxCode,
			Percent:   t.Percent,
		})
	}

	lines := make([]domain.InvoiceLine, 0, len(out.Lines))
	for _, l := range out.Lines {
		lines = append(lines, domain.InvoiceLine{
			LineID:          l.LineID,
			ItemCode:        l.ItemCode,
			ItemDescription: l.ItemDescription,
			Quantity:        l.Quantity,
			UnitCode:        l.UnitCode,
			UnitPrice:       l.UnitPrice,
			LineExtension:   l.LineExtension,
			TaxAmount:       l.TaxAmount,
		})
	}

	invoice := &domain.InvoiceDocument{
		ProfileID:        out.ProfileID,
		InvoiceID:        out.InvoiceID,
		IssueDate:        out.IssueDate,
		IssueTime:        out.IssueTime,
		DueDate:          out.DueDate,
		CurrencyCode:     out.CurrencyCode,
		CUFE:             out.CUFE,
		PaymentMeansCode: out.PaymentMeansCode,
		Issuer: domain.Party{
			Name:  out.Issuer.Name,
			TaxID: out.Issuer.TaxID,
		},
		Receiver: domain.Party{
			Name:  out.Receiver.Name,
			TaxID: out.Receiver.TaxID,
		},
		TaxTotals:      taxTotals,
		LineExtension:  out.LineExtension,
		TaxExclusive:   out.TaxExclusive,
		TaxInclusive:   out.TaxInclusive,
		AllowanceTotal: out.AllowanceTotal,
		PayableAmount:  out.PayableAmount,
		Lines:          lines,
	}

	return invoice, nil
}

func firstCandidateText(payload geminiResponse) string {
	for _, c := range payload.Candidates {
		for _, p := range c.Content.Parts {
			if strings.TrimSpace(p.Text) != "" {
				return p.Text
			}
		}
	}
	return ""
}

func stripCodeFence(v string) string {
	trim := strings.TrimSpace(v)
	if !strings.HasPrefix(trim, "```") {
		return trim
	}
	trim = strings.TrimPrefix(trim, "```")
	trim = strings.TrimPrefix(trim, "json")
	trim = strings.TrimSpace(trim)
	trim = strings.TrimSuffix(trim, "```")
	return strings.TrimSpace(trim)
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	Temperature      int            `json:"temperature"`
	ResponseMimeType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseSchema"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type llmInvoiceOutput struct {
	ProfileID        string `json:"profile_id"`
	InvoiceID        string `json:"invoice_id"`
	IssueDate        string `json:"issue_date"`
	IssueTime        string `json:"issue_time"`
	DueDate          string `json:"due_date"`
	CurrencyCode     string `json:"currency_code"`
	CUFE             string `json:"cufe"`
	PaymentMeansCode string `json:"payment_means_code"`
	Issuer           struct {
		Name  string `json:"name"`
		TaxID string `json:"tax_id"`
	} `json:"issuer"`
	Receiver struct {
		Name  string `json:"name"`
		TaxID string `json:"tax_id"`
	} `json:"receiver"`
	TaxTotals []struct {
		TaxAmount float64 `json:"tax_amount"`
		Taxable   float64 `json:"taxable"`
		TaxCode   string  `json:"tax_code"`
		Percent   float64 `json:"percent"`
	} `json:"tax_totals"`
	LineExtension  float64 `json:"line_extension"`
	TaxExclusive   float64 `json:"tax_exclusive"`
	TaxInclusive   float64 `json:"tax_inclusive"`
	AllowanceTotal float64 `json:"allowance_total"`
	PayableAmount  float64 `json:"payable_amount"`
	Lines          []struct {
		LineID          string  `json:"line_id"`
		ItemCode        string  `json:"item_code"`
		ItemDescription string  `json:"item_description"`
		Quantity        float64 `json:"quantity"`
		UnitCode        string  `json:"unit_code"`
		UnitPrice       float64 `json:"unit_price"`
		LineExtension   float64 `json:"line_extension"`
		TaxAmount       float64 `json:"tax_amount"`
	} `json:"lines"`
}

var geminiResponseSchema = map[string]any{
	"type":     "object",
	"required": []string{"cufe", "issuer", "receiver", "lines"},
	"properties": map[string]any{
		"profile_id":         map[string]any{"type": "string"},
		"invoice_id":         map[string]any{"type": "string"},
		"issue_date":         map[string]any{"type": "string"},
		"issue_time":         map[string]any{"type": "string"},
		"due_date":           map[string]any{"type": "string"},
		"currency_code":      map[string]any{"type": "string"},
		"cufe":               map[string]any{"type": "string"},
		"payment_means_code": map[string]any{"type": "string"},
		"issuer": map[string]any{
			"type":     "object",
			"required": []string{"name", "tax_id"},
			"properties": map[string]any{
				"name":   map[string]any{"type": "string"},
				"tax_id": map[string]any{"type": "string"},
			},
		},
		"receiver": map[string]any{
			"type":     "object",
			"required": []string{"name", "tax_id"},
			"properties": map[string]any{
				"name":   map[string]any{"type": "string"},
				"tax_id": map[string]any{"type": "string"},
			},
		},
		"tax_totals": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tax_amount": map[string]any{"type": "number"},
					"taxable":    map[string]any{"type": "number"},
					"tax_code":   map[string]any{"type": "string"},
					"percent":    map[string]any{"type": "number"},
				},
			},
		},
		"line_extension":  map[string]any{"type": "number"},
		"tax_exclusive":   map[string]any{"type": "number"},
		"tax_inclusive":   map[string]any{"type": "number"},
		"allowance_total": map[string]any{"type": "number"},
		"payable_amount":  map[string]any{"type": "number"},
		"lines": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":     "object",
				"required": []string{"line_id", "item_description"},
				"properties": map[string]any{
					"line_id":          map[string]any{"type": "string"},
					"item_code":        map[string]any{"type": "string"},
					"item_description": map[string]any{"type": "string"},
					"quantity":         map[string]any{"type": "number"},
					"unit_code":        map[string]any{"type": "string"},
					"unit_price":       map[string]any{"type": "number"},
					"line_extension":   map[string]any{"type": "number"},
					"tax_amount":       map[string]any{"type": "number"},
				},
			},
		},
	},
}

const prompt = `Eres un extractor de facturas electrónicas colombianas. Analiza el PDF y responde SOLO JSON estricto siguiendo el schema. Incluye CUFE/UUID, emisor, receptor, fecha de vencimiento, códigos de pago, impuestos, descuentos (allowance_total) y lineas de detalle.`

var _ ports.InvoiceLLMExtractor = (*GeminiExtractor)(nil)
