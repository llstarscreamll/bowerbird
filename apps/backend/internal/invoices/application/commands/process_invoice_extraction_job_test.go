package commands

import (
	"archive/zip"
	"bytes"
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

func (r *fakeInvoiceRepo) ExistsBySource(ctx context.Context, sourceName string, sourceID string) (bool, error) {
	return r.messageProcessed, nil
}

func (r *fakeInvoiceRepo) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return r.cufeExists, nil
}

func (r *fakeInvoiceRepo) GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error) {
	return nil, nil, nil
}

func (r *fakeInvoiceRepo) ListInvoices(ctx context.Context, limit int, query string) ([]domain.InvoiceHeaderRecord, bool, error) {
	return nil, false, nil
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
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": []byte("zipdata")}}
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{}
	repo := &fakeInvoiceRepo{messageProcessed: true}

	uc := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo)
	err := uc.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "inbox-message",
		SourceID:   "msg-1",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 0)
}

func TestExtractProcessesZIPPDFAndPersistsInvoice(t *testing.T) {
	archive := makeTestZip(t, map[string][]byte{
		"inv.pdf": []byte("%PDF-1.4 file"),
	})
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": archive}}
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

	uc := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo)
	uc.create.newID = func() string { return "id_1" }
	err := uc.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-1",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 1, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 1)
}

func TestExtractSkipsWhenZIPHasNoSupportedFiles(t *testing.T) {
	archive := makeTestZip(t, map[string][]byte{"ignored.txt": []byte("notes")})
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": archive}}
	xmlExtractor := &fakeXMLExtractor{}
	llmExtractor := &fakeLLMExtractor{}
	repo := &fakeInvoiceRepo{}

	uc := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo)
	err := uc.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-2",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 0)
}

func TestExtractSkipsWhenCUFEAlreadyExists(t *testing.T) {
	archive := makeTestZip(t, map[string][]byte{"inv.pdf": []byte("%PDF-1.4 file")})
	store := &fakeExtractFileStore{data: map[string][]byte{"k1": archive}}
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
	repo := &fakeInvoiceRepo{cufeExists: true}

	uc := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo)
	err := uc.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-3",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 0)
}

func makeTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip file: %v", err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write zip file: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buf.Bytes()
}
