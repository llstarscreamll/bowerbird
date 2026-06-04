package application

import (
	"context"
	"fmt"
	"log/slog"

	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type InvoiceRepository interface {
	ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error)
	ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error)
	domain.InvoiceWriteRepository
}

type ExtractInvoiceStatus string

const (
	ExtractInvoiceStatusReady   ExtractInvoiceStatus = "ready"
	ExtractInvoiceStatusSkipped ExtractInvoiceStatus = "skipped"
)

type ExtractInvoiceSkipReason string

const (
	SkipReasonMessageAlreadyProcessed ExtractInvoiceSkipReason = "message_already_processed"
	SkipReasonNoSupportedDocument     ExtractInvoiceSkipReason = "no_supported_document"
	SkipReasonCUFEAlreadyExists       ExtractInvoiceSkipReason = "cufe_already_exists"
	SkipReasonExtractionFailed        ExtractInvoiceSkipReason = "extraction_failed"
)

type ExtractInvoiceResult struct {
	Status     ExtractInvoiceStatus
	SkipReason ExtractInvoiceSkipReason
	Source     string
	HeaderID   string
	Invoice    *domain.InvoiceDocument
}

type ExtractInvoiceCommand struct {
	store        platformStorage.FileStore
	classifier   domain.DocumentClassifier
	xmlExtractor domain.InvoiceXMLExtractor
	llmExtractor domain.InvoiceLLMExtractor
	repo         InvoiceRepository
	persist      *PersistInvoiceCommand
	logger       *slog.Logger
}

func NewExtractInvoiceCommand(
	store platformStorage.FileStore,
	xmlExtractor domain.InvoiceXMLExtractor,
	llmExtractor domain.InvoiceLLMExtractor,
	repo InvoiceRepository,
) *ExtractInvoiceCommand {
	return &ExtractInvoiceCommand{
		store:        store,
		classifier:   domain.NewInvoiceDocumentClassifier(),
		xmlExtractor: xmlExtractor,
		llmExtractor: llmExtractor,
		repo:         repo,
		persist:      NewPersistInvoiceCommand(repo),
		logger:       slog.Default(),
	}
}

func (cmd *ExtractInvoiceCommand) Execute(ctx context.Context, input contractEvents.InvoiceExtractionRequested) (*ExtractInvoiceResult, error) {
	if cmd.store == nil {
		return nil, fmt.Errorf("file store is required")
	}
	if cmd.repo == nil {
		return nil, fmt.Errorf("invoice repository is required")
	}
	if cmd.persist == nil {
		return nil, fmt.Errorf("persist invoice command is required")
	}

	attachments, err := cmd.downloadAttachments(ctx, input.AttachmentRefs)
	if err != nil {
		return nil, err
	}
	if len(attachments) == 0 {
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonNoSupportedDocument}, nil
	}

	classification, err := cmd.classifier.ClassifyAttachments(attachments)
	if err != nil {
		return nil, fmt.Errorf("classify attachments: %w", err)
	}

	processed, err := cmd.repo.ExistsInvoiceBySourceMessageID(ctx, input.SourceMessageID)
	if err != nil {
		return nil, fmt.Errorf("check invoice by source message id: %w", err)
	}
	if processed {
		cmd.logger.Info("invoice extraction skipped by source message", "source_message_id", input.SourceMessageID)
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonMessageAlreadyProcessed}, nil
	}

	foundDuplicate := false
	for _, group := range classification.Groups {
		invoice, source, documentRefS3Key, err := cmd.extractInvoiceDocument(ctx, group)
		if err != nil {
			cmd.logger.Warn("invoice extraction failed for group", "group_key", group.GroupKey, "error", err)
			continue
		}
		if invoice == nil {
			continue
		}

		duplicated, err := cmd.repo.ExistsInvoiceByCUFE(ctx, invoice.CUFE)
		if err != nil {
			return nil, fmt.Errorf("check invoice by cufe: %w", err)
		}
		if duplicated {
			cmd.logger.Info("invoice extraction skipped by cufe", "cufe", invoice.CUFE)
			foundDuplicate = true
			continue
		}

		persisted, err := cmd.persist.Execute(ctx, PersistInvoiceInput{
			SourceMessageID:  input.SourceMessageID,
			ExtractionSource: source,
			DocumentRefS3Key: documentRefS3Key,
			Invoice:          invoice,
		})
		if err != nil {
			return nil, fmt.Errorf("persist invoice: %w", err)
		}

		cmd.logger.Info("invoice extracted and persisted", "source", source, "cufe", invoice.CUFE, "header_id", persisted.HeaderID)
		return &ExtractInvoiceResult{
			Status:   ExtractInvoiceStatusReady,
			Source:   source,
			HeaderID: persisted.HeaderID,
			Invoice:  invoice,
		}, nil
	}

	if len(classification.Groups) == 0 {
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonNoSupportedDocument}, nil
	}
	if foundDuplicate {
		return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonCUFEAlreadyExists}, nil
	}

	return &ExtractInvoiceResult{Status: ExtractInvoiceStatusSkipped, SkipReason: SkipReasonExtractionFailed}, nil
}

func (cmd *ExtractInvoiceCommand) downloadAttachments(ctx context.Context, refs []contractEvents.AttachmentRef) ([]domain.AttachmentContent, error) {
	attachments := make([]domain.AttachmentContent, 0, len(refs))
	for _, ref := range refs {
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

	return attachments, nil
}

func (cmd *ExtractInvoiceCommand) extractInvoiceDocument(ctx context.Context, group domain.DocumentGroup) (*domain.InvoiceDocument, string, string, error) {
	if !group.SupportsInvoiceExtraction() {
		return nil, "", "", nil
	}

	source := group.PreferredDocumentSource()
	if source == "xml" {
		if cmd.xmlExtractor == nil {
			return nil, "", "", fmt.Errorf("xml extractor is required")
		}
		invoice, err := cmd.xmlExtractor.ParseInvoiceXML(group.XML.Data)
		if err != nil {
			return nil, "", "", fmt.Errorf("extract invoice from xml: %w", err)
		}
		return invoice, "xml", group.XML.S3Key, nil
	}

	if source == "llm" {
		if cmd.llmExtractor == nil {
			return nil, "", "", fmt.Errorf("llm extractor is required")
		}
		invoice, err := cmd.llmExtractor.ExtractFromPDF(ctx, group.PDF.Data)
		if err != nil {
			return nil, "", "", fmt.Errorf("extract invoice from pdf with llm: %w", err)
		}
		return invoice, "llm", group.PDF.S3Key, nil
	}

	return nil, "", "", nil
}
