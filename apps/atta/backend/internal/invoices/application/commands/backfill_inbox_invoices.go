package commands

import (
	"context"
	"log/slog"
	"time"

	contractEvents "github.com/atta/internal/contracts/events"
	inboxapi "github.com/atta/internal/inbox/api"
	"github.com/atta/internal/invoices/application/ports"
	contractJobs "github.com/atta/internal/invoices/contracts/jobs"
	"github.com/atta/internal/platform/id"
	"github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/tenant"
)

const inboxBackfillPageSize = 50

type BackfillInboxInvoicesCommand struct {
	source    inboxapi.InvoiceBackfillSource
	enqueue   *CreateInvoicesFromInboxMessageCommand
	jobQueue  jobs.TaskQueue
	receivers ports.ReceiverDirectory
	logger    *slog.Logger
	now       func() time.Time
	newID     func() string
}

func NewBackfillInboxInvoicesCommand(
	source inboxapi.InvoiceBackfillSource,
	enqueue *CreateInvoicesFromInboxMessageCommand,
	jobQueue jobs.TaskQueue,
	receivers ports.ReceiverDirectory,
) *BackfillInboxInvoicesCommand {
	if source == nil {
		panic("inbox backfill source is required")
	}
	if enqueue == nil {
		panic("inbox invoice enqueue command is required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}
	if receivers == nil {
		panic("receiver directory is required")
	}
	return &BackfillInboxInvoicesCommand{
		source:    source,
		enqueue:   enqueue,
		jobQueue:  jobQueue,
		receivers: receivers,
		logger:    slog.Default(),
		now:       time.Now,
		newID:     id.NewULID,
	}
}

func (cmd *BackfillInboxInvoicesCommand) Enqueue(ctx context.Context) error {
	hasReceiver, err := cmd.receivers.HasAny(ctx)
	if err != nil {
		return err
	}
	if !hasReceiver {
		return nil
	}
	job := contractJobs.InboxBackfillJob{
		ID:       cmd.newID(),
		QueuedAt: cmd.now().UTC().Format(time.RFC3339Nano),
	}
	payload, err := contractJobs.MarshalInboxBackfillRequested(job)
	if err != nil {
		return err
	}
	return cmd.jobQueue.Enqueue(ctx, jobs.Job{
		Type:    contractJobs.InvoiceInboxBackfillRequestedType,
		Payload: payload,
	})
}

func (cmd *BackfillInboxInvoicesCommand) Execute(ctx context.Context, job contractJobs.InboxBackfillJob) error {
	hasReceiver, err := cmd.receivers.HasAny(ctx)
	if err != nil {
		return err
	}
	if !hasReceiver {
		return nil
	}

	page, err := cmd.source.ListExtractionCandidates(ctx, job.Cursor, inboxBackfillPageSize)
	if err != nil {
		return err
	}

	tenantSlug, _ := tenant.TenantIDFromContext(ctx)
	for _, item := range page.Items {
		event := contractEvents.InboxMessageReceived{
			EventID:           cmd.newID(),
			TenantID:          tenantSlug,
			AccountID:         "backfill",
			Provider:          "backfill",
			ProviderMessageID: item.MessageID,
			MessageInternalID: item.MessageID,
			Subject:           item.Subject,
			Body:              item.Snippet,
			OccurredAt:        cmd.now().UTC().Format(time.RFC3339Nano),
		}
		for _, att := range item.Attachments {
			event.AttachmentRefs = append(event.AttachmentRefs, contractEvents.AttachmentRef{
				S3Key:    att.S3Key,
				Filename: att.Filename,
				MimeType: att.MimeType,
			})
		}
		if err := cmd.enqueue.Execute(ctx, event); err != nil {
			return err
		}
	}

	if page.NextCursor == "" {
		cmd.logger.Info("invoice inbox backfill page complete", "count", len(page.Items))
		return nil
	}

	next := contractJobs.InboxBackfillJob{
		ID:       cmd.newID(),
		Cursor:   page.NextCursor,
		QueuedAt: cmd.now().UTC().Format(time.RFC3339Nano),
	}
	payload, err := contractJobs.MarshalInboxBackfillRequested(next)
	if err != nil {
		return err
	}
	return cmd.jobQueue.Enqueue(ctx, jobs.Job{
		Type:    contractJobs.InvoiceInboxBackfillRequestedType,
		Payload: payload,
	})
}
