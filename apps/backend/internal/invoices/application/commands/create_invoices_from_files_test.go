package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type createFilesRepoStub struct {
	alreadyProcessed bool
	cufeExists       bool
	persistedHeaders []domain.InvoiceHeaderRecord
}

func (r *createFilesRepoStub) ExistsBySource(ctx context.Context, sourceName string, sourceID string) (bool, error) {
	return r.alreadyProcessed, nil
}

func (r *createFilesRepoStub) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return r.cufeExists, nil
}

func (r *createFilesRepoStub) GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error) {
	return nil, nil, nil
}

func (r *createFilesRepoStub) ListInvoices(ctx context.Context, limit int, query string) ([]domain.InvoiceHeaderRecord, bool, error) {
	return nil, false, nil
}

func (r *createFilesRepoStub) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	r.persistedHeaders = append(r.persistedHeaders, header)
	return nil
}

func (r *createFilesRepoStub) ApplyCatalogLinking(ctx context.Context, headerID string, issuerPartyID *string, linkingStatus string, lines []ports.LineLinkUpdate) error {
	return nil
}

type createFilesStoreStub struct {
	data map[string][]byte
}

func (s *createFilesStoreStub) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *createFilesStoreStub) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	payload, ok := s.data[input.Path]
	if !ok {
		return nil, errors.New("not found")
	}

	return payload, nil
}

func (s *createFilesStoreStub) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	_, ok := s.data[input.Path]
	return ok, nil
}

func (s *createFilesStoreStub) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (s *createFilesStoreStub) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, errors.New("not implemented")
}

func (s *createFilesStoreStub) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, errors.New("not implemented")
}

type createFilesXMLExtractorStub struct {
	called  int
	invoice *domain.InvoiceDocument
	err     error
}

func (e *createFilesXMLExtractorStub) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
	e.called++
	if e.err != nil {
		return nil, e.err
	}

	return e.invoice, nil
}

type createFilesLLMExtractorStub struct {
	called  int
	invoice *domain.InvoiceDocument
	err     error
}

func (e *createFilesLLMExtractorStub) ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error) {
	e.called++
	if e.err != nil {
		return nil, e.err
	}

	return e.invoice, nil
}

func TestCreateInvoicesFromFilesSkipsAlreadyProcessedSource(t *testing.T) {
	store := &createFilesStoreStub{data: map[string][]byte{"k1": []byte("zipdata")}}
	xmlExtractor := &createFilesXMLExtractorStub{}
	llmExtractor := &createFilesLLMExtractorStub{}
	repo := &createFilesRepoStub{alreadyProcessed: true}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
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

func TestCreateInvoicesFromFilesSkipsUnsupportedInputFiles(t *testing.T) {
	store := &createFilesStoreStub{data: map[string][]byte{"k1": []byte("some text content")}}
	xmlExtractor := &createFilesXMLExtractorStub{invoice: validInvoiceDoc("CUFE-1")}
	llmExtractor := &createFilesLLMExtractorStub{}
	repo := &createFilesRepoStub{}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-1",
		Files:      []contractJobs.File{{Path: "k1", Filename: "notes.txt", MimeType: "text/plain"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 0)
}

func TestCreateInvoicesFromFilesProcessesZipAndPersistsInvoice(t *testing.T) {
	archive := makeCreateFilesZip(t, map[string][]byte{"inv.pdf": []byte("%PDF-1.4 file")})
	store := &createFilesStoreStub{data: map[string][]byte{"k1": archive}}
	xmlExtractor := &createFilesXMLExtractorStub{}
	llmExtractor := &createFilesLLMExtractorStub{invoice: validInvoiceDoc("CUFE-LLM")}
	repo := &createFilesRepoStub{}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	cmd.create.newID = func() string { return "id_1" }
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-1",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, xmlExtractor.called)
	assert.Equal(t, 1, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 1)
	assert.Equal(t, "files-uploaded-by-user", repo.persistedHeaders[0].SourceName)
	assert.Equal(t, "upload-zip-1", repo.persistedHeaders[0].SourceID)
}

func TestCreateInvoicesFromFilesNormalizesXMLRawDataToJSON(t *testing.T) {
	archive := makeCreateFilesZip(t, map[string][]byte{"inv.xml": []byte("<Invoice>raw</Invoice>")})
	store := &createFilesStoreStub{data: map[string][]byte{"k1": archive}}
	xmlInvoice := validInvoiceDoc("CUFE-XML")
	xmlInvoice.RawData = []byte("<Invoice>raw</Invoice>")
	xmlExtractor := &createFilesXMLExtractorStub{invoice: xmlInvoice}
	llmExtractor := &createFilesLLMExtractorStub{}
	repo := &createFilesRepoStub{}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	cmd.create.newID = func() string { return "id_1" }
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-xml-1",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, xmlExtractor.called)
	assert.Equal(t, 0, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 1)
	assert.True(t, json.Valid(repo.persistedHeaders[0].RawData))

	var rawAsString string
	require.NoError(t, json.Unmarshal(repo.persistedHeaders[0].RawData, &rawAsString))
	assert.Equal(t, "<Invoice>raw</Invoice>", rawAsString)
}

func TestCreateInvoicesFromFilesSkipsZipWithoutSupportedFiles(t *testing.T) {
	archive := makeCreateFilesZip(t, map[string][]byte{"ignored.txt": []byte("notes")})
	store := &createFilesStoreStub{data: map[string][]byte{"k1": archive}}
	xmlExtractor := &createFilesXMLExtractorStub{}
	llmExtractor := &createFilesLLMExtractorStub{}
	repo := &createFilesRepoStub{}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
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

func TestCreateInvoicesFromFilesSkipsWhenCUFEAlreadyExists(t *testing.T) {
	archive := makeCreateFilesZip(t, map[string][]byte{"inv.pdf": []byte("%PDF-1.4 file")})
	store := &createFilesStoreStub{data: map[string][]byte{"k1": archive}}
	xmlExtractor := &createFilesXMLExtractorStub{}
	llmExtractor := &createFilesLLMExtractorStub{invoice: validInvoiceDoc("CUFE-LLM")}
	repo := &createFilesRepoStub{cufeExists: true}

	cmd := NewCreateInvoicesFromFilesCommand(store, xmlExtractor, llmExtractor, repo, nil, NewCreateInvoiceCommand(repo, nil, nil))
	err := cmd.Execute(context.Background(), contractJobs.ExtractInvoicesFromFilesJob{
		ID:         "job-1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload-zip-3",
		Files:      []contractJobs.File{{Path: "k1", Filename: "bundle.zip", MimeType: "application/zip"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, llmExtractor.called)
	assert.Len(t, repo.persistedHeaders, 0)
}

func validInvoiceDoc(cufe string) *domain.InvoiceDocument {
	return &domain.InvoiceDocument{
		CUFE:          cufe,
		InvoiceID:     "INV-1",
		Issuer:        domain.Party{Name: "Issuer", TaxID: "123"},
		Receiver:      domain.Party{Name: "Receiver", TaxID: "456"},
		Lines:         []domain.InvoiceLine{{LineID: "1", ItemDescription: "x", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		CurrencyCode:  "COP",
		PayableAmount: 10,
	}
}

func makeCreateFilesZip(t *testing.T, files map[string][]byte) []byte {
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
