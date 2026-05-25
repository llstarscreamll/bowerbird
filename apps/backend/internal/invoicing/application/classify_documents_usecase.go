package application

import (
	"context"
	"fmt"
	"log/slog"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/observability"
)

type AttachmentContentReader interface {
	ReadByKey(ctx context.Context, key string) ([]byte, error)
}

type DocumentKind = domain.DocumentKind

const (
	DocumentKindZIP   = domain.DocumentKindZIP
	DocumentKindXML   = domain.DocumentKindXML
	DocumentKindPDF   = domain.DocumentKindPDF
	DocumentKindOther = domain.DocumentKindOther
)

type ClassifiedDocument = domain.ClassifiedDocument
type DocumentGroup = domain.DocumentGroup
type ClassificationResult = domain.ClassificationResult

type ClassifyDocumentsUseCase struct {
	reader     AttachmentContentReader
	classifier domain.DocumentClassifier
	logger     *slog.Logger
	metrics    observability.Metrics
}

func NewClassifyDocumentsUseCase(reader AttachmentContentReader) *ClassifyDocumentsUseCase {
	return &ClassifyDocumentsUseCase{
		reader:     reader,
		classifier: domain.NewInvoiceDocumentClassifier(),
		logger:     slog.Default(),
		metrics:    observability.NoopMetrics{},
	}
}

func (u *ClassifyDocumentsUseCase) ClassifyFromInboxEvent(ctx context.Context, event contractevents.InboxMessageReceived) (*ClassificationResult, error) {
	if u.reader == nil {
		return nil, fmt.Errorf("attachment content reader is required")
	}

	attachments := make([]domain.AttachmentContent, 0, len(event.AttachmentRefs))
	for _, ref := range event.AttachmentRefs {
		if ref.S3Key == "" {
			continue
		}

		data, err := u.reader.ReadByKey(ctx, ref.S3Key)
		if err != nil {
			return nil, fmt.Errorf("read attachment from s3 key %s: %w", ref.S3Key, err)
		}

		attachments = append(attachments, domain.AttachmentContent{
			Filename: ref.Filename,
			S3Key:    ref.S3Key,
			Data:     data,
		})
	}

	result, err := u.classifier.ClassifyAttachments(attachments)
	if err != nil {
		return nil, fmt.Errorf("classify attachments: %w", err)
	}

	u.metrics.IncCounter("invoicing_documents_classified_total", map[string]string{"tenant_slug": event.TenantSlug, "result": "done"})
	u.logger.Info(
		"invoicing documents classified",
		"tenant_slug",
		event.TenantSlug,
		"groups",
		len(result.Groups),
		"unclassified",
		len(result.Unclassified),
		"compressed_sources",
		result.CompressedSources,
	)

	return result, nil
}
