package handlers

import (
	"context"
	"testing"

	awsEvents "github.com/aws/aws-lambda-go/events"
	invoicingCommands "github.com/bowerbird/internal/invoices/application/commands"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
)

type processorFileStore struct{}

func (s *processorFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}

func (s *processorFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	return []byte("<Invoice><ID>INV-1</ID><UUID>CUFE-1</UUID></Invoice>"), nil
}

func (s *processorFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	return true, nil
}

func (s *processorFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (s *processorFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, nil
}

func (s *processorFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, nil
}

type processorRepo struct{}

func (r *processorRepo) ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error) {
	return false, nil
}

func (r *processorRepo) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return false, nil
}

func (r *processorRepo) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	return nil
}

type processorXMLExtractor struct{}

func (e *processorXMLExtractor) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
	return &domain.InvoiceDocument{
		CUFE:          "CUFE-1",
		InvoiceID:     "INV-1",
		Issuer:        domain.Party{Name: "Issuer", CompanyID: "123"},
		Receiver:      domain.Party{Name: "Receiver", CompanyID: "456"},
		CurrencyCode:  "COP",
		PayableAmount: 10,
		Lines:         []domain.InvoiceLine{{LineID: "1", ItemDescription: "x", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
	}, nil
}

type processorLLMExtractor struct{}

func (e *processorLLMExtractor) ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error) {
	return nil, nil
}

func TestProcessInvoiceExtractionRequestedHandlesMessage(t *testing.T) {
	cmd := invoicingCommands.NewProcessInvoiceExtractionJobCommand(&processorFileStore{}, &processorXMLExtractor{}, &processorLLMExtractor{}, &processorRepo{})
	processor := NewProcessInvoiceExtractionRequested(cmd)

	detail, err := contractJobs.MarshalInvoiceExtractionRequested(contractJobs.InvoiceExtractionRequested{
		JobID:  "job_1",
		Source: "msg_1",
		Files: []contractJobs.File{
			{Path: "k1", Filename: "factura.xml"},
		},
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	ctx := tenant.WithTenantID(context.Background(), "tenant_1")
	err = processor.HandleSQS(ctx, awsEvents.SQSMessage{Body: string(detail)})
	if err != nil {
		t.Fatalf("handle message failed: %v", err)
	}
}

func TestProcessInvoiceExtractionRequestedRequiresTenantInContext(t *testing.T) {
	cmd := invoicingCommands.NewProcessInvoiceExtractionJobCommand(&processorFileStore{}, &processorXMLExtractor{}, &processorLLMExtractor{}, &processorRepo{})
	processor := NewProcessInvoiceExtractionRequested(cmd)

	detail, err := contractJobs.MarshalInvoiceExtractionRequested(contractJobs.InvoiceExtractionRequested{
		JobID:  "job_1",
		Source: "msg_1",
		Files: []contractJobs.File{
			{Path: "k1", Filename: "factura.xml"},
		},
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	err = processor.HandleSQS(context.Background(), awsEvents.SQSMessage{Body: string(detail)})
	if err == nil {
		t.Fatal("expected tenant id error")
	}
}
