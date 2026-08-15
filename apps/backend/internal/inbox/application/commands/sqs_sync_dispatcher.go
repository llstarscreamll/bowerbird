package commands

import (
	"context"
	"fmt"

	inboxJobs "github.com/bowerbird/internal/inbox/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
)

type SQSSyncAccountJobDispatcher struct {
	queue jobs.Queue
}

func NewSQSSyncAccountJobDispatcher(queue jobs.Queue) *SQSSyncAccountJobDispatcher {
	if queue == nil {
		panic("job queue is required")
	}
	return &SQSSyncAccountJobDispatcher{queue: queue}
}

func (d *SQSSyncAccountJobDispatcher) DispatchSyncAccount(ctx context.Context, job SyncAccountJob) error {
	payload, err := inboxJobs.MarshalInboxSyncAccount(inboxJobs.InboxSyncAccountJob{
		TenantID:  job.TenantID,
		AccountID: job.AccountID,
		Provider:  job.Provider,
	})
	if err != nil {
		return fmt.Errorf("marshal sync account job: %w", err)
	}

	return d.queue.Dispatch(ctx, jobs.Job{
		Type:    inboxJobs.InboxSyncAccountType,
		Payload: payload,
	})
}
