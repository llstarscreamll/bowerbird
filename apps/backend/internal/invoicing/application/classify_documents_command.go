package application

import (
	"context"
	"fmt"
	"log/slog"

	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
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

type ClassifyDocumentsCommand struct {
	store      platformStorage.FileStore
	classifier domain.DocumentClassifier
	logger     *slog.Logger
}

func NewClassifyDocumentsCommand(store platformStorage.FileStore) *ClassifyDocumentsCommand {
	return &ClassifyDocumentsCommand{
		store:      store,
		classifier: domain.NewInvoiceDocumentClassifier(),
		logger:     slog.Default(),
	}
}

func (cmd *ClassifyDocumentsCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) (*ClassificationResult, error) {
	if cmd.store == nil {
		return nil, fmt.Errorf("file store is required")
	}

	attachments := make([]domain.AttachmentContent, 0, len(event.AttachmentRefs))
	for _, ref := range event.AttachmentRefs {
		if ref.S3Key == "" {
			continue
		}

		data, err := cmd.store.ReadFile(ctx, platformStorage.ReadFileInput{Path: ref.S3Key})
		if err != nil {
			return nil, fmt.Errorf("read attachment from key %s: %w", ref.S3Key, err)
		}

		attachments = append(attachments, domain.AttachmentContent{
			Filename: ref.Filename,
			S3Key:    ref.S3Key,
			Data:     data,
		})
	}

	result, err := cmd.classifier.ClassifyAttachments(attachments)
	if err != nil {
		return nil, fmt.Errorf("classify attachments: %w", err)
	}

	cmd.logger.Info(
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
