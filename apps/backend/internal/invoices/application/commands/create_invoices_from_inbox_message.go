package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

type CreateInvoicesFromInboxMessageCommand struct {
	jobQueue jobs.Queue
	logger   *slog.Logger
	now      func() time.Time
	newID    func() string
}

func NewCreateInvoicesFromInboxMessageCommand(jobQueue jobs.Queue) *CreateInvoicesFromInboxMessageCommand {
	return &CreateInvoicesFromInboxMessageCommand{
		jobQueue: jobQueue,
		logger:   slog.Default(),
		now:      time.Now,
		newID:    id.NewULID,
	}
}

func (cmd *CreateInvoicesFromInboxMessageCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) error {
	if !hasInvoiceKeyword(event.Subject, event.Body) {
		cmd.logger.Info("invoicing event skipped: missing invoice keyword", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	if !hasSupportedAttachment(event.AttachmentRefs) {
		cmd.logger.Info("invoicing event skipped: missing supported attachments", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	if cmd.jobQueue == nil {
		cmd.logger.Info("invoicing candidate detected but queue not configured", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
		return nil
	}

	job := contractJobs.InvoiceExtractionRequested{
		JobID:    cmd.newID(),
		Source:   "inbox-message",
		Files:    mapAttachmentRefs(event.AttachmentRefs),
		QueuedAt: cmd.now().UTC().Format(time.RFC3339Nano),
	}

	payload, err := contractJobs.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return err
	}

	err = cmd.jobQueue.Dispatch(ctx, jobs.Job{
		Type:    contractJobs.InvoiceExtractionRequestedType,
		Payload: payload,
	})
	if err != nil {
		return err
	}

	cmd.logger.Info("invoice extraction job queued", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID, "attachments", len(event.AttachmentRefs))
	return nil
}

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

func mapAttachmentRefs(refs []contractEvents.AttachmentRef) []contractJobs.File {
	mapped := make([]contractJobs.File, 0, len(refs))
	for _, ref := range refs {
		mapped = append(mapped, contractJobs.File{
			Path:     ref.S3Key,
			Filename: ref.Filename,
			MimeType: ref.MimeType,
		})
	}

	return mapped
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

type CreateInvoiceInput struct {
	SourceMessageID  string
	ExtractionSource string
	DocumentRefS3Key string
	Invoice          *domain.InvoiceDocument
}

type CreateInvoiceResult struct {
	HeaderID string
	LineIDs  []string
}

type CreateInvoiceCommand struct {
	repo   ports.InvoiceWriteRepository
	logger *slog.Logger
	now    func() time.Time
	newID  func() string
}

func NewCreateInvoiceCommand(repo ports.InvoiceWriteRepository) *CreateInvoiceCommand {
	return &CreateInvoiceCommand{repo: repo, logger: slog.Default(), now: time.Now, newID: id.NewULID}
}

func (cmd *CreateInvoiceCommand) Execute(ctx context.Context, input CreateInvoiceInput) (*CreateInvoiceResult, error) {
	if cmd.repo == nil {
		return nil, fmt.Errorf("invoice write repository is required")
	}
	if input.Invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}
	if err := input.Invoice.Validate(); err != nil {
		return nil, err
	}

	now := cmd.now().UTC()
	headerID := cmd.newID()
	headerRawData := input.Invoice.RawData
	if len(headerRawData) == 0 {
		headerRawData = []byte("{}")
	}

	header := domain.InvoiceHeaderRecord{
		ID:               headerID,
		SourceMessageID:  input.SourceMessageID,
		CUFE:             input.Invoice.CUFE,
		InvoiceNumber:    input.Invoice.InvoiceID,
		IssuerName:       input.Invoice.Issuer.Name,
		IssuerTaxID:      input.Invoice.Issuer.CompanyID,
		ReceiverName:     input.Invoice.Receiver.Name,
		ReceiverTaxID:    input.Invoice.Receiver.CompanyID,
		CurrencyCode:     input.Invoice.CurrencyCode,
		IssueDate:        input.Invoice.IssueDateTimeUTC(),
		PaymentCode:      input.Invoice.PaymentMeansCode,
		Subtotal:         input.Invoice.LineExtension,
		TaxTotal:         input.Invoice.TaxAmountTotal(),
		GrandTotal:       input.Invoice.PayableAmount,
		DocumentRefS3Key: input.DocumentRefS3Key,
		ExtractionSource: input.ExtractionSource,
		RawData:          headerRawData,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	lines := make([]domain.InvoiceLineRecord, 0, len(input.Invoice.Lines))
	lineIDs := make([]string, 0, len(input.Invoice.Lines))
	for idx, line := range input.Invoice.Lines {
		lineID := cmd.newID()
		lineNumber := line.NumberOrDefault(idx + 1)
		lineRawData, err := json.Marshal(line)
		if err != nil {
			return nil, fmt.Errorf("marshal invoice line raw data: %w", err)
		}

		lines = append(lines, domain.InvoiceLineRecord{
			ID:              lineID,
			InvoiceHeaderID: headerID,
			LineNumber:      lineNumber,
			ItemCode:        "",
			Description:     line.ItemDescription,
			Quantity:        line.Quantity,
			UnitPrice:       line.UnitPrice,
			LineTaxTotal:    line.TaxAmount,
			LineTotal:       line.LineExtension,
			RawData:         lineRawData,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
		lineIDs = append(lineIDs, lineID)
	}

	if err := cmd.repo.PersistInvoiceAtomic(ctx, header, lines); err != nil {
		return nil, err
	}
	cmd.logger.Info("invoice persisted atomically", "header_id", headerID, "cufe", header.CUFE, "lines", len(lines))

	return &CreateInvoiceResult{HeaderID: headerID, LineIDs: lineIDs}, nil
}

func hasSupportedAttachment(refs []contractEvents.AttachmentRef) bool {
	for _, ref := range refs {
		ext := strings.ToLower(filepath.Ext(ref.Filename))
		if ext == ".xml" || ext == ".pdf" || ext == ".zip" {
			return true
		}

		mime := strings.ToLower(strings.TrimSpace(ref.MimeType))
		if strings.Contains(mime, "pdf") || strings.Contains(mime, "xml") || strings.Contains(mime, "zip") {
			return true
		}
	}

	return false
}

func hasInvoiceKeyword(subject, body string) bool {
	combined := strings.ToLower(strings.TrimSpace(subject + "\n" + body))
	if combined == "" {
		return false
	}

	keywords := []string{
		"factura electronica",
		"facturación electrónica",
		"factura electrónica",
		"facturacion electronica",
		"facturacion",
		"factura",
		"invoice",
	}

	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}
