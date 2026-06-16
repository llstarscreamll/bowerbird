package commands

import (
	"context"
	"errors"
	"testing"

	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (s *fakeExtractFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExtractFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	payload, ok := s.data[input.Path]
	if !ok {
		return nil, errors.New("not found")
	}
	return payload, nil
}

func (s *fakeExtractFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	_, ok := s.data[input.Path]
	return ok, nil
}

func (s *fakeExtractFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (s *fakeExtractFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExtractFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
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

	uc := NewProcessInvoiceExtractionJobCommand(store, xmlExtractor, llmExtractor, repo)
	res, err := uc.Execute(context.Background(), contractJobs.InvoiceExtractionRequested{
		JobID:  "job-1",
		Source: "msg-1",
		Files: []contractJobs.File{
			{Path: "k1", Filename: "inv.xml"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, ProcessInvoiceExtractionJobStatusSkipped, res.Status)
	assert.Equal(t, SkipReasonMessageAlreadyProcessed, res.SkipReason)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
}

func TestExtractUsesXMLFirstAndSkipsWhenCUFEExists(t *testing.T) {
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": []byte("<Invoice></Invoice>")}}
	xmlExtractor := &fakeXMLExtractor{invoice: &domain.InvoiceDocument{CUFE: "CUFE-1"}}
	llmExtractor := &fakeLLMExtractor{invoice: &domain.InvoiceDocument{CUFE: "LLM-CUFE"}}
	repo := &fakeInvoiceRepo{cufeExists: true}

	uc := NewProcessInvoiceExtractionJobCommand(store, xmlExtractor, llmExtractor, repo)
	res, err := uc.Execute(context.Background(), contractJobs.InvoiceExtractionRequested{
		JobID:  "job-1",
		Source: "msg-1",
		Files: []contractJobs.File{
			{Path: "k1", Filename: "inv.xml"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, ProcessInvoiceExtractionJobStatusSkipped, res.Status)
	assert.Equal(t, SkipReasonCUFEAlreadyExists, res.SkipReason)
	assert.Equal(t, 1, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
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

	uc := NewProcessInvoiceExtractionJobCommand(store, xmlExtractor, llmExtractor, repo)
	uc.create.newID = func() string { return "id_1" }
	res, err := uc.Execute(context.Background(), contractJobs.InvoiceExtractionRequested{
		JobID:  "job-1",
		Source: "msg-1",
		Files: []contractJobs.File{
			{Path: "k1", Filename: "inv.pdf"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, ProcessInvoiceExtractionJobStatusReady, res.Status)
	assert.Equal(t, "llm", res.Source)
	require.NotNil(t, res.Invoice)
	assert.Equal(t, 1, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 1)
}
