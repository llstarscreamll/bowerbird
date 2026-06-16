package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

type ProcessInvoiceExtractionJobStatus string

const (
	ProcessInvoiceExtractionJobStatusReady   ProcessInvoiceExtractionJobStatus = "ready"
	ProcessInvoiceExtractionJobStatusSkipped ProcessInvoiceExtractionJobStatus = "skipped"
)

type ProcessInvoiceExtractionJobSkipReason string

const (
	SkipReasonMessageAlreadyProcessed ProcessInvoiceExtractionJobSkipReason = "message_already_processed"
	SkipReasonNoSupportedDocument     ProcessInvoiceExtractionJobSkipReason = "no_supported_document"
	SkipReasonCUFEAlreadyExists       ProcessInvoiceExtractionJobSkipReason = "cufe_already_exists"
	SkipReasonExtractionFailed        ProcessInvoiceExtractionJobSkipReason = "extraction_failed"
)

type ProcessInvoiceExtractionJobResult struct {
	Status     ProcessInvoiceExtractionJobStatus
	SkipReason ProcessInvoiceExtractionJobSkipReason
	Source     string
	HeaderID   string
	Invoice    *domain.InvoiceDocument
}

type ProcessInvoiceExtractionJobCommand struct {
	fileStore    platformStorage.FileStore
	classifier   domain.DocumentClassifier
	xmlExtractor ports.InvoiceXMLExtractor
	llmExtractor ports.InvoiceLLMExtractor
	repo         ports.InvoiceRepository
	create       *CreateInvoiceCommand
	logger       *slog.Logger
}

func NewProcessInvoiceExtractionJobCommand(
	fileStore platformStorage.FileStore,
	xmlExtractor ports.InvoiceXMLExtractor,
	llmExtractor ports.InvoiceLLMExtractor,
	repo ports.InvoiceRepository,
) *ProcessInvoiceExtractionJobCommand {
	if fileStore == nil {
		panic("file store is required")
	}
	if xmlExtractor == nil {
		panic("xml extractor is required")
	}
	if llmExtractor == nil {
		panic("llm extractor is required")
	}
	if repo == nil {
		panic("invoice repository is required")
	}

	return &ProcessInvoiceExtractionJobCommand{
		fileStore:    fileStore,
		classifier:   domain.NewInvoiceDocumentClassifier(),
		xmlExtractor: xmlExtractor,
		llmExtractor: llmExtractor,
		repo:         repo,
		create:       NewCreateInvoiceCommand(repo),
		logger:       slog.Default(),
	}
}

func (cmd *ProcessInvoiceExtractionJobCommand) Execute(ctx context.Context, input contractJobs.InvoiceExtractionRequested) (*ProcessInvoiceExtractionJobResult, error) {
	attachments, err := cmd.downloadAttachments(ctx, input.Files)
	if err != nil {
		return nil, err
	}
	if len(attachments) == 0 {
		return &ProcessInvoiceExtractionJobResult{Status: ProcessInvoiceExtractionJobStatusSkipped, SkipReason: SkipReasonNoSupportedDocument}, nil
	}

	classification, err := cmd.classifier.ClassifyAttachments(attachments)
	if err != nil {
		return nil, fmt.Errorf("classify attachments: %w", err)
	}

	processed, err := cmd.repo.ExistsInvoiceBySourceMessageID(ctx, input.Source)
	if err != nil {
		return nil, fmt.Errorf("check invoice by source message id: %w", err)
	}
	if processed {
		cmd.logger.Info("invoice extraction skipped by source message", "source_message_id", input.Source)
		return &ProcessInvoiceExtractionJobResult{Status: ProcessInvoiceExtractionJobStatusSkipped, SkipReason: SkipReasonMessageAlreadyProcessed}, nil
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

		persisted, err := cmd.create.Execute(ctx, CreateInvoiceInput{
			SourceMessageID:  input.Source,
			ExtractionSource: source,
			DocumentRefS3Key: documentRefS3Key,
			Invoice:          invoice,
		})
		if err != nil {
			return nil, fmt.Errorf("persist invoice: %w", err)
		}

		cmd.logger.Info("invoice extracted and persisted", "source", source, "cufe", invoice.CUFE, "header_id", persisted.HeaderID)
		return &ProcessInvoiceExtractionJobResult{
			Status:   ProcessInvoiceExtractionJobStatusReady,
			Source:   source,
			HeaderID: persisted.HeaderID,
			Invoice:  invoice,
		}, nil
	}

	if len(classification.Groups) == 0 {
		return &ProcessInvoiceExtractionJobResult{Status: ProcessInvoiceExtractionJobStatusSkipped, SkipReason: SkipReasonNoSupportedDocument}, nil
	}
	if foundDuplicate {
		return &ProcessInvoiceExtractionJobResult{Status: ProcessInvoiceExtractionJobStatusSkipped, SkipReason: SkipReasonCUFEAlreadyExists}, nil
	}

	return &ProcessInvoiceExtractionJobResult{Status: ProcessInvoiceExtractionJobStatusSkipped, SkipReason: SkipReasonExtractionFailed}, nil
}

func (cmd *ProcessInvoiceExtractionJobCommand) downloadAttachments(ctx context.Context, refs []contractJobs.File) ([]domain.AttachmentContent, error) {
	attachments := make([]domain.AttachmentContent, 0, len(refs))
	for _, ref := range refs {
		if ref.Path == "" {
			continue
		}

		data, err := cmd.fileStore.ReadFile(ctx, platformStorage.ReadFileInput{Path: ref.Path})
		if err != nil {
			return nil, fmt.Errorf("read attachment from key %s: %w", ref.Path, err)
		}

		attachments = append(attachments, domain.AttachmentContent{
			Filename: ref.Filename,
			S3Key:    ref.Path,
			Data:     data,
		})
	}

	return attachments, nil
}

func (cmd *ProcessInvoiceExtractionJobCommand) extractInvoiceDocument(ctx context.Context, group domain.DocumentGroup) (*domain.InvoiceDocument, string, string, error) {
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
