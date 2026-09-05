package commands

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
)

type CreateInvoicesFromInboxMessageCommand struct {
	jobQueue jobs.TaskQueue
	logger   *slog.Logger
	now      func() time.Time
	newID    func() string
}

func NewCreateInvoicesFromInboxMessageCommand(jobQueue jobs.TaskQueue) *CreateInvoicesFromInboxMessageCommand {
	if jobQueue == nil {
		panic("job queue is required")
	}

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

	job := contractJobs.ExtractInvoicesFromFilesJob{
		ID:         cmd.newID(),
		SourceName: "inbox-message",
		SourceID:   event.MessageInternalID,
		Files:      mapAttachmentRefs(event.AttachmentRefs),
		QueuedAt:   cmd.now().UTC().Format(time.RFC3339Nano),
	}

	payload, err := contractJobs.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return err
	}

	err = cmd.jobQueue.Enqueue(ctx, jobs.Job{
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
