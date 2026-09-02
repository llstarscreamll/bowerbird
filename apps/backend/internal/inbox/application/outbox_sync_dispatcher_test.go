package application_test

import (
	"context"
	"testing"

	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxJobs "github.com/bowerbird/internal/inbox/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeJobQueue struct {
	jobs []jobs.Job
}

func (q *fakeJobQueue) Enqueue(ctx context.Context, job jobs.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func TestOutboxSyncAccountJobDispatcher_EnqueuesInboxSyncAccount(t *testing.T) {
	queue := &fakeJobQueue{}
	dispatcher := inboxCommands.NewOutboxSyncAccountJobDispatcher(queue)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	require.NoError(t, dispatcher.DispatchSyncAccount(ctx, inboxCommands.SyncAccountJob{
		TenantID:  "tenant-a",
		AccountID: "acc-1",
		Provider:  "gmail",
	}))

	require.Len(t, queue.jobs, 1)
	assert.Equal(t, inboxJobs.InboxSyncAccountType, queue.jobs[0].Type)

	decoded, err := inboxJobs.UnmarshalInboxSyncAccount(queue.jobs[0].Payload)
	require.NoError(t, err)
	assert.Equal(t, "tenant-a", decoded.TenantID)
	assert.Equal(t, "acc-1", decoded.AccountID)
	assert.Equal(t, "gmail", decoded.Provider)
}
