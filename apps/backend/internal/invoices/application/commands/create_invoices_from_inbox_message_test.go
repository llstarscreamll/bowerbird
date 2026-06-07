package commands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	contractevents "github.com/bowerbird/internal/contracts/events"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBusinessPublisher struct {
	jobs []jobs.Job
}

func (p *fakeBusinessPublisher) Dispatch(ctx context.Context, job jobs.Job) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestCheckQueuesInvoiceExtractionJob(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCreateInvoicesFromInboxMessageCommand(publisher)
	uc.newID = func() string { return "evt_1" }

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_src_1",
		TenantID:          "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "provider_msg_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica de mayo",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "factura.pdf"},
		},
	})
	require.NoError(t, err)
	require.Len(t, publisher.jobs, 1)
	assert.Equal(t, contractJobs.InvoiceExtractionRequestedType, publisher.jobs[0].Type)

	var queued contractJobs.InvoiceExtractionRequested
	require.NoError(t, json.Unmarshal(publisher.jobs[0].Payload, &queued))
	assert.Equal(t, "inbox-message", queued.Source)
}

func TestCheckSkipsNonCandidates(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCreateInvoicesFromInboxMessageCommand(publisher)

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantID:          "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		Subject:           "meeting notes from vendor",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "notes.txt"},
		},
	})
	require.NoError(t, err)
	assert.Len(t, publisher.jobs, 0)
}

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

type fakeInvoiceWriteRepo struct {
	called bool
	header domain.InvoiceHeaderRecord
	lines  []domain.InvoiceLineRecord
}

func (r *fakeInvoiceWriteRepo) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	r.called = true
	r.header = header
	r.lines = lines
	return nil
}

func TestCreateInvoiceCommandBuildsAtomicRecords(t *testing.T) {
	repo := &fakeInvoiceWriteRepo{}
	uc := NewCreateInvoiceCommand(repo)
	uc.now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	ids := []string{"hdr_1", "line_1", "line_2"}
	i := 0
	uc.newID = func() string {
		id := ids[i]
		i++
		return id
	}

	res, err := uc.Execute(context.Background(), CreateInvoiceInput{
		SourceMessageID:  "msg_1",
		ExtractionSource: "xml",
		DocumentRefS3Key: "tenant/t/inbox/.../invoice.xml",
		Invoice: &domain.InvoiceDocument{
			CUFE:             "CUFE-1",
			InvoiceID:        "FE-1",
			IssueDate:        "2026-05-25",
			IssueTime:        "10:00:00-05:00",
			CurrencyCode:     "COP",
			PaymentMeansCode: "1",
			Issuer:           domain.Party{Name: "Proveedor", CompanyID: "900"},
			Receiver:         domain.Party{Name: "Cliente", CompanyID: "901"},
			LineExtension:    100,
			TaxTotals:        []domain.TaxTotal{{TaxAmount: 19}, {TaxAmount: 1}},
			PayableAmount:    120,
			RawData:          []byte(`{"src":"xml"}`),
			Lines: []domain.InvoiceLine{
				{LineID: "1", ItemDescription: "Servicio A", Quantity: 1, UnitPrice: 50, LineExtension: 50, TaxAmount: 9.5},
				{LineID: "2", ItemDescription: "Servicio B", Quantity: 1, UnitPrice: 50, LineExtension: 50, TaxAmount: 10.5},
			},
		},
	})
	require.NoError(t, err)
	assert.True(t, repo.called)
	assert.Equal(t, "CUFE-1", repo.header.CUFE)
	assert.Equal(t, 20.0, repo.header.TaxTotal)
	require.Len(t, repo.lines, 2)
	assert.Equal(t, 1, repo.lines[0].LineNumber)
	assert.Equal(t, 2, repo.lines[1].LineNumber)
	assert.Equal(t, "hdr_1", res.HeaderID)
	assert.Len(t, res.LineIDs, 2)
}
