package application

import (
	"context"
	"fmt"
	"log/slog"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

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
	store      platformstorage.FileStore
	classifier domain.DocumentClassifier
	logger     *slog.Logger
}

func NewClassifyDocumentsUseCase(store platformstorage.FileStore) *ClassifyDocumentsUseCase {
	return &ClassifyDocumentsUseCase{
		store:      store,
		classifier: domain.NewInvoiceDocumentClassifier(),
		logger:     slog.Default(),
	}
}

func (u *ClassifyDocumentsUseCase) ClassifyFromInboxEvent(ctx context.Context, event contractevents.InboxMessageReceived) (*ClassificationResult, error) {
	if u.store == nil {
		return nil, fmt.Errorf("file store is required")
	}

	attachments := make([]domain.AttachmentContent, 0, len(event.AttachmentRefs))
	for _, ref := range event.AttachmentRefs {
		if ref.S3Key == "" {
			continue
		}

		data, err := u.store.ReadFile(ctx, platformstorage.ReadFileInput{Path: ref.S3Key})
		if err != nil {
			return nil, fmt.Errorf("read attachment from key %s: %w", ref.S3Key, err)
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
