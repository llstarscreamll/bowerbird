package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bowerbird/internal/invoices/domain"
)

func TestGeminiExtractorExtractFromPDFSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Fatalf("expected api key query param, got %q", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		genCfg, ok := req["generationConfig"].(map[string]any)
		if !ok {
			t.Fatalf("missing generationConfig")
		}
		if genCfg["responseMimeType"] != "application/json" {
			t.Fatalf("expected responseMimeType application/json")
		}

		_, _ = w.Write([]byte(`{
			"candidates":[
				{"content":{"parts":[{"text":"{\"cufe\":\"CUFE-1\",\"issuer\":{\"name\":\"Proveedor\",\"tax_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"tax_id\":\"901\"},\"due_date\":\"2026-08-08\",\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}],\"tax_totals\":[{\"tax_amount\":19,\"taxable\":100,\"tax_code\":\"01\",\"percent\":19}],\"allowance_total\":50,\"payable_amount\":119}"}]}}
			]
		}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor(GeminiExtractorConfig{
		APIKey:     "test-key",
		Model:      "gemini-test",
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new extractor failed: %v", err)
	}

	doc, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if doc.CUFE != "CUFE-1" {
		t.Fatalf("expected CUFE, got %q", doc.CUFE)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(doc.Lines))
	}
	if doc.DueDate != "2026-08-08" {
		t.Fatalf("expected due date, got %q", doc.DueDate)
	}
	if doc.AllowanceTotal != 50 {
		t.Fatalf("expected allowance total 50, got %v", doc.AllowanceTotal)
	}
}

func TestGeminiExtractorExtractFromPDFFailsOnMissingCUFE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"issuer\":{\"name\":\"Proveedor\",\"tax_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"tax_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}]}"}]}}]}`))
	}))
	defer server.Close()

	extractor, _ := NewGeminiExtractor(GeminiExtractorConfig{APIKey: "test-key", Model: "gemini-test", Endpoint: server.URL, HTTPClient: server.Client()})
	_, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err == nil || !errorsIs(err, domain.ErrMissingCUFE) {
		t.Fatalf("expected missing CUFE error, got %v", err)
	}
}

func TestGeminiExtractorRejectsUnknownJSONFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"cufe\":\"CUFE-1\",\"issuer\":{\"name\":\"Proveedor\",\"tax_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"tax_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}],\"unexpected\":123}"}]}}]}`))
	}))
	defer server.Close()

	extractor, _ := NewGeminiExtractor(GeminiExtractorConfig{APIKey: "test-key", Model: "gemini-test", Endpoint: server.URL, HTTPClient: server.Client()})
	_, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err == nil || !strings.Contains(err.Error(), "decode structured invoice output") {
		t.Fatalf("expected strict decode error, got %v", err)
	}
}

func TestGeminiExtractorExtractFromPDFRetriesOn429AndThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if call < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"1ms"}]}}`))
			return
		}

		_, _ = w.Write([]byte(`{
			"candidates":[
				{"content":{"parts":[{"text":"{\"cufe\":\"CUFE-1\",\"issuer\":{\"name\":\"Proveedor\",\"tax_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"tax_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}],\"tax_totals\":[{\"tax_amount\":19,\"taxable\":100,\"tax_code\":\"01\",\"percent\":19}],\"payable_amount\":119}"}]}}
			]
		}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor(GeminiExtractorConfig{
		APIKey:      "test-key",
		Model:       "gemini-test",
		Endpoint:    server.URL,
		HTTPClient:  server.Client(),
		BaseBackoff: time.Millisecond,
		MaxBackoff:  2 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new extractor failed: %v", err)
	}

	doc, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if doc.CUFE != "CUFE-1" {
		t.Fatalf("expected CUFE-1, got %q", doc.CUFE)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestGeminiExtractorExtractFromPDFDoesNotRetryHardQuota429(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"Quota exceeded ... limit: 0"}}`))
	}))
	defer server.Close()

	extractor, err := NewGeminiExtractor(GeminiExtractorConfig{
		APIKey:      "test-key",
		Model:       "gemini-test",
		Endpoint:    server.URL,
		HTTPClient:  server.Client(),
		BaseBackoff: time.Millisecond,
		MaxBackoff:  2 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new extractor failed: %v", err)
	}

	_, err = extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err == nil || !strings.Contains(err.Error(), "status=429") {
		t.Fatalf("expected 429 error, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 attempt, got %d", got)
	}
}

func errorsIs(err, target error) bool {
	if err == nil {
		return target == nil
	}
	if err == target {
		return true
	}
	type causer interface{ Unwrap() error }
	if c, ok := err.(causer); ok {
		return errorsIs(c.Unwrap(), target)
	}
	return false
}
