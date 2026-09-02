package commands

import (
	"context"
	"fmt"

	inboxJobs "github.com/bowerbird/internal/inbox/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
)

type OutboxSyncAccountJobDispatcher struct {
	queue jobs.TaskQueue
}

func NewOutboxSyncAccountJobDispatcher(queue jobs.TaskQueue) *OutboxSyncAccountJobDispatcher {
	if queue == nil {
		panic("task queue is required")
	}
	return &OutboxSyncAccountJobDispatcher{queue: queue}
}

func (d *OutboxSyncAccountJobDispatcher) DispatchSyncAccount(ctx context.Context, job SyncAccountJob) error {
	payload, err := inboxJobs.MarshalInboxSyncAccount(inboxJobs.InboxSyncAccountJob{
		TenantID:  job.TenantID,
		AccountID: job.AccountID,
		Provider:  job.Provider,
	})
	if err != nil {
		return fmt.Errorf("marshal sync account job: %w", err)
	}

	return d.queue.Enqueue(ctx, jobs.Job{
		Type:    inboxJobs.InboxSyncAccountType,
		Payload: payload,
	})
}
