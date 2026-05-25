package application

import (
	"context"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

type fakeDedupRepo struct {
	messageProcessed bool
	cufeExists       bool
}

func (r *fakeDedupRepo) ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error) {
	return r.messageProcessed, nil
}

func (r *fakeDedupRepo) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return r.cufeExists, nil
}

type fakeXMLExtractor struct {
	called  int
	invoice *domain.InvoiceDocument
	err     error
}

func (e *fakeXMLExtractor) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
	e.called++
	if e.err != nil {
		return nil, e.err
	}
	return e.invoice, nil
}

type fakeLLMExtractor struct {
	called  int
	invoice *domain.InvoiceDocument
	err     error
}

func (e *fakeLLMExtractor) ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error) {
	e.called++
	if e.err != nil {
		return nil, e.err
	}
	return e.invoice, nil
}

func TestExtractFromGroupSkipsWhenMessageAlreadyProcessed(t *testing.T) {
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{}
	repo := &fakeDedupRepo{messageProcessed: true}

	uc := NewExtractInvoiceUseCase(xmlExtractor, llmExtractor, repo)
	res, err := uc.ExtractFromGroup(context.Background(), "msg-1", domain.DocumentGroup{})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if res.Status != ExtractInvoiceStatusSkipped || res.SkipReason != SkipReasonMessageAlreadyProcessed {
		t.Fatalf("unexpected result: %#v", res)
	}
	if xmlExtractor.called != 0 || llmExtractor.called != 0 {
		t.Fatalf("expected no extractor calls when message already processed")
	}
}

func TestExtractFromGroupUsesXMLFirstAndSkipsWhenCUFEExists(t *testing.T) {
	xmlExtractor := &fakeXMLExtractor{invoice: &domain.InvoiceDocument{CUFE: "CUFE-1"}}
	llmExtractor := &fakeLLMExtractor{invoice: &domain.InvoiceDocument{CUFE: "LLM-CUFE"}}
	repo := &fakeDedupRepo{cufeExists: true}

	uc := NewExtractInvoiceUseCase(xmlExtractor, llmExtractor, repo)
	res, err := uc.ExtractFromGroup(context.Background(), "msg-1", domain.DocumentGroup{
		XML: &domain.ClassifiedDocument{Kind: domain.DocumentKindXML, Data: []byte("<xml/>")},
		PDF: &domain.ClassifiedDocument{Kind: domain.DocumentKindPDF, Data: []byte("%PDF")},
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if res.Status != ExtractInvoiceStatusSkipped || res.SkipReason != SkipReasonCUFEAlreadyExists {
		t.Fatalf("unexpected result: %#v", res)
	}
	if xmlExtractor.called != 1 {
		t.Fatalf("expected xml extractor called once, got %d", xmlExtractor.called)
	}
	if llmExtractor.called != 0 {
		t.Fatalf("expected llm extractor not called when xml exists")
	}
}

func TestExtractFromGroupFallsBackToLLMAndReturnsReady(t *testing.T) {
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{invoice: &domain.InvoiceDocument{CUFE: "CUFE-LLM"}}
	repo := &fakeDedupRepo{}

	uc := NewExtractInvoiceUseCase(xmlExtractor, llmExtractor, repo)
	res, err := uc.ExtractFromGroup(context.Background(), "msg-1", domain.DocumentGroup{
		PDF: &domain.ClassifiedDocument{Kind: domain.DocumentKindPDF, Data: []byte("%PDF")},
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if res.Status != ExtractInvoiceStatusReady || res.Source != "llm" || res.Invoice == nil {
		t.Fatalf("unexpected result: %#v", res)
	}
	if llmExtractor.called != 1 {
		t.Fatalf("expected llm extractor called once, got %d", llmExtractor.called)
	}
}
