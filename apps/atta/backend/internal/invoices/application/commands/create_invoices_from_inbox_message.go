package commands

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	contractEvents "github.com/atta/internal/contracts/events"
	"github.com/atta/internal/invoices/application/ports"
	contractJobs "github.com/atta/internal/invoices/contracts/jobs"
	"github.com/atta/internal/platform/id"
	"github.com/atta/internal/platform/jobs"
)

type CreateInvoicesFromInboxMessageCommand struct {
	jobQueue  jobs.TaskQueue
	receivers ports.ReceiverDirectory
	logger    *slog.Logger
	now       func() time.Time
	newID     func() string
}

func NewCreateInvoicesFromInboxMessageCommand(jobQueue jobs.TaskQueue, receivers ports.ReceiverDirectory) *CreateInvoicesFromInboxMessageCommand {
	if jobQueue == nil {
		panic("job queue is required")
	}
	if receivers == nil {
		panic("receiver directory is required")
	}

	return &CreateInvoicesFromInboxMessageCommand{
		jobQueue:  jobQueue,
		receivers: receivers,
		logger:    slog.Default(),
		now:       time.Now,
		newID:     id.NewULID,
	}
}

func (cmd *CreateInvoicesFromInboxMessageCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) error {
	hasReceiver, err := cmd.receivers.HasAny(ctx)
	if err != nil {
		return err
	}
	if !hasReceiver {
		cmd.logger.Info("invoicing event skipped: no legal entity", "tenant_slug", event.TenantID, "message_id", event.MessageInternalID)
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
