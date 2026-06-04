package application

import (
	"context"
	"errors"
	"testing"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type fakeInvoiceRepo struct {
	messageProcessed bool
	cufeExists       bool
	persistedHeaders []domain.InvoiceHeaderRecord
}

func (r *fakeInvoiceRepo) ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error) {
	return r.messageProcessed, nil
}

func (r *fakeInvoiceRepo) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return r.cufeExists, nil
}

func (r *fakeInvoiceRepo) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	r.persistedHeaders = append(r.persistedHeaders, header)
	return nil
}

type fakeExtractFileStore struct {
	data map[string][]byte
}

func (s *fakeExtractFileStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExtractFileStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	payload, ok := s.data[input.Path]
	if !ok {
		return nil, errors.New("not found")
	}
	return payload, nil
}

func (s *fakeExtractFileStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	_, ok := s.data[input.Path]
	return ok, nil
}

func (s *fakeExtractFileStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	return nil
}

func (s *fakeExtractFileStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExtractFileStore) PresignDownload(ctx context.Context, input platformstorage.PresignDownloadInput) (*platformstorage.PresignDownloadResult, error) {
	return nil, errors.New("not implemented")
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

func TestExtractSkipsWhenMessageAlreadyProcessed(t *testing.T) {
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": []byte("<Invoice></Invoice>")}}
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{}
	repo := &fakeInvoiceRepo{messageProcessed: true}

	uc := NewExtractInvoiceCommand(store, xmlExtractor, llmExtractor, repo)
	res, err := uc.Execute(context.Background(), contractevents.InvoiceExtractionRequested{
		EventID:         "evt-1",
		TenantSlug:      "tenant-1",
		SourceMessageID: "msg-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "k1", Filename: "inv.xml"},
		},
	})
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

func TestExtractUsesXMLFirstAndSkipsWhenCUFEExists(t *testing.T) {
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": []byte("<Invoice></Invoice>")}}
	xmlExtractor := &fakeXMLExtractor{invoice: &domain.InvoiceDocument{CUFE: "CUFE-1"}}
	llmExtractor := &fakeLLMExtractor{invoice: &domain.InvoiceDocument{CUFE: "LLM-CUFE"}}
	repo := &fakeInvoiceRepo{cufeExists: true}

	uc := NewExtractInvoiceCommand(store, xmlExtractor, llmExtractor, repo)
	res, err := uc.Execute(context.Background(), contractevents.InvoiceExtractionRequested{
		EventID:         "evt-1",
		TenantSlug:      "tenant-1",
		SourceMessageID: "msg-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "k1", Filename: "inv.xml"},
		},
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

func TestExtractFallsBackToLLMAndReturnsReady(t *testing.T) {
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": []byte("%PDF-1.4 file")}}
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{invoice: &domain.InvoiceDocument{
		CUFE:          "CUFE-LLM",
		InvoiceID:     "INV-1",
		Issuer:        domain.Party{Name: "Issuer", CompanyID: "123"},
		Receiver:      domain.Party{Name: "Receiver", CompanyID: "456"},
		Lines:         []domain.InvoiceLine{{LineID: "1", ItemDescription: "x", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		CurrencyCode:  "COP",
		PayableAmount: 10,
	}}
	repo := &fakeInvoiceRepo{}

	uc := NewExtractInvoiceCommand(store, xmlExtractor, llmExtractor, repo)
	uc.persist.newID = func() string { return "id_1" }
	res, err := uc.Execute(context.Background(), contractevents.InvoiceExtractionRequested{
		EventID:         "evt-1",
		TenantSlug:      "tenant-1",
		SourceMessageID: "msg-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "k1", Filename: "inv.pdf"},
		},
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
	if len(repo.persistedHeaders) != 1 {
		t.Fatalf("expected one persisted header, got %d", len(repo.persistedHeaders))
	}
}
