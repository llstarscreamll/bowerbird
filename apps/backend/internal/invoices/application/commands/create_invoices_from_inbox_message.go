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
