package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
				{"content":{"parts":[{"text":"{\"cufe\":\"CUFE-1\",\"issuer\":{\"name\":\"Proveedor\",\"company_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"company_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}],\"tax_totals\":[{\"tax_amount\":19,\"taxable\":100,\"tax_code\":\"01\",\"percent\":19}],\"payable_amount\":119}"}]}}
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
}

func TestGeminiExtractorExtractFromPDFFailsOnMissingCUFE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"issuer\":{\"name\":\"Proveedor\",\"company_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"company_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}]}"}]}}]}`))
	}))
	defer server.Close()

	extractor, _ := NewGeminiExtractor(GeminiExtractorConfig{APIKey: "test-key", Endpoint: server.URL, HTTPClient: server.Client()})
	_, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err == nil || !errorsIs(err, domain.ErrMissingCUFE) {
		t.Fatalf("expected missing CUFE error, got %v", err)
	}
}

func TestGeminiExtractorRejectsUnknownJSONFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"cufe\":\"CUFE-1\",\"issuer\":{\"name\":\"Proveedor\",\"company_id\":\"900\"},\"receiver\":{\"name\":\"Cliente\",\"company_id\":\"901\"},\"lines\":[{\"line_id\":\"1\",\"item_description\":\"Servicio\"}],\"unexpected\":123}"}]}}]}`))
	}))
	defer server.Close()

	extractor, _ := NewGeminiExtractor(GeminiExtractorConfig{APIKey: "test-key", Endpoint: server.URL, HTTPClient: server.Client()})
	_, err := extractor.ExtractFromPDF(context.Background(), []byte("%PDF-1.4 sample"))
	if err == nil || !strings.Contains(err.Error(), "decode structured invoice output") {
		t.Fatalf("expected strict decode error, got %v", err)
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
