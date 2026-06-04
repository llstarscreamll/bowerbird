package events

import (
	"context"
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingapp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type subscriberFileStore struct{}

func (s *subscriberFileStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}

func (s *subscriberFileStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	return []byte("<Invoice><ID>INV-1</ID><UUID>CUFE-1</UUID></Invoice>"), nil
}

func (s *subscriberFileStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	return true, nil
}

func (s *subscriberFileStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	return nil
}

func (s *subscriberFileStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	return nil, nil
}

func (s *subscriberFileStore) PresignDownload(ctx context.Context, input platformstorage.PresignDownloadInput) (*platformstorage.PresignDownloadResult, error) {
	return nil, nil
}

type subscriberRepo struct{}

func (r *subscriberRepo) ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error) {
	return false, nil
}

func (r *subscriberRepo) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	return false, nil
}

func (r *subscriberRepo) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	return nil
}

type subscriberXMLExtractor struct{}

func (e *subscriberXMLExtractor) ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error) {
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

func TestInvoiceExtractionRequestedSubscriberHandlesEvent(t *testing.T) {
	cmd := invoicingapp.NewExtractInvoiceCommand(&subscriberFileStore{}, &subscriberXMLExtractor{}, nil, &subscriberRepo{})
	subscriber := NewInvoiceExtractionRequestedSubscriber(cmd)

	detail, err := contractevents.MarshalInvoiceExtractionRequested(contractevents.InvoiceExtractionRequested{
		EventID:         "evt_1",
		TenantSlug:      "tenant_1",
		SourceMessageID: "msg_1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "k1", Filename: "factura.xml"},
		},
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	err = subscriber.HandleEventBridge(context.Background(), awsevents.CloudWatchEvent{
		DetailType: contractevents.InvoiceExtractionRequestedDetailType,
		Detail:     detail,
	})
	if err != nil {
		t.Fatalf("handle event failed: %v", err)
	}
}
