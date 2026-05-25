package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

const defaultGeminiEndpoint = "https://generativelanguage.googleapis.com"

type GeminiExtractor struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

type GeminiExtractorConfig struct {
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

func NewGeminiExtractor(cfg GeminiExtractorConfig) (*GeminiExtractor, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gemini-2.0-flash"
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultGeminiEndpoint
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &GeminiExtractor{
		apiKey:     cfg.APIKey,
		model:      model,
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: httpClient,
	}, nil
}

func (e *GeminiExtractor) ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error) {
	if len(pdfData) == 0 {
		return nil, fmt.Errorf("pdf data is required")
	}

	body, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{
				{Text: geminiPrompt},
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini response status=%d body=%s", resp.StatusCode, string(respData))
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
	if strings.TrimSpace(invoice.Issuer.Name) == "" || strings.TrimSpace(invoice.Issuer.CompanyID) == "" {
		return nil, domain.ErrMissingIssuer
	}
	if strings.TrimSpace(invoice.Receiver.Name) == "" || strings.TrimSpace(invoice.Receiver.CompanyID) == "" {
		return nil, domain.ErrMissingReceiver
	}
	if len(invoice.Lines) == 0 {
		return nil, domain.ErrMissingLineItems
	}

	invoice.RawData = []byte(structured)
	return invoice, nil
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
		CurrencyCode:     out.CurrencyCode,
		CUFE:             out.CUFE,
		PaymentMeansCode: out.PaymentMeansCode,
		Issuer: domain.Party{
			Name:      out.Issuer.Name,
			CompanyID: out.Issuer.CompanyID,
		},
		Receiver: domain.Party{
			Name:      out.Receiver.Name,
			CompanyID: out.Receiver.CompanyID,
		},
		TaxTotals:     taxTotals,
		LineExtension: out.LineExtension,
		TaxExclusive:  out.TaxExclusive,
		TaxInclusive:  out.TaxInclusive,
		PayableAmount: out.PayableAmount,
		Lines:         lines,
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
	CurrencyCode     string `json:"currency_code"`
	CUFE             string `json:"cufe"`
	PaymentMeansCode string `json:"payment_means_code"`
	Issuer           struct {
		Name      string `json:"name"`
		CompanyID string `json:"company_id"`
	} `json:"issuer"`
	Receiver struct {
		Name      string `json:"name"`
		CompanyID string `json:"company_id"`
	} `json:"receiver"`
	TaxTotals []struct {
		TaxAmount float64 `json:"tax_amount"`
		Taxable   float64 `json:"taxable"`
		TaxCode   string  `json:"tax_code"`
		Percent   float64 `json:"percent"`
	} `json:"tax_totals"`
	LineExtension float64 `json:"line_extension"`
	TaxExclusive  float64 `json:"tax_exclusive"`
	TaxInclusive  float64 `json:"tax_inclusive"`
	PayableAmount float64 `json:"payable_amount"`
	Lines         []struct {
		LineID          string  `json:"line_id"`
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
		"currency_code":      map[string]any{"type": "string"},
		"cufe":               map[string]any{"type": "string"},
		"payment_means_code": map[string]any{"type": "string"},
		"issuer": map[string]any{
			"type":     "object",
			"required": []string{"name", "company_id"},
			"properties": map[string]any{
				"name":       map[string]any{"type": "string"},
				"company_id": map[string]any{"type": "string"},
			},
		},
		"receiver": map[string]any{
			"type":     "object",
			"required": []string{"name", "company_id"},
			"properties": map[string]any{
				"name":       map[string]any{"type": "string"},
				"company_id": map[string]any{"type": "string"},
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
		"line_extension": map[string]any{"type": "number"},
		"tax_exclusive":  map[string]any{"type": "number"},
		"tax_inclusive":  map[string]any{"type": "number"},
		"payable_amount": map[string]any{"type": "number"},
		"lines": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":     "object",
				"required": []string{"line_id", "item_description"},
				"properties": map[string]any{
					"line_id":          map[string]any{"type": "string"},
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

const geminiPrompt = `Eres un extractor de facturas electronicas colombianas. Analiza el PDF y responde SOLO JSON estricto siguiendo el schema. Incluye CUFE/UUID, emisor, receptor, codigos de pago, impuestos y lineas de detalle.`

var _ domain.InvoiceLLMExtractor = (*GeminiExtractor)(nil)
