package events_test

import (
	"context"
	"testing"

	contractevents "github.com/bowerbird/internal/contracts/events"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	eventsV1 "github.com/bowerbird/internal/inbox/presentation/events"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/require"
)

type fakeTaskQueue struct {
	jobs []jobs.Job
}

func (q *fakeTaskQueue) Enqueue(_ context.Context, job jobs.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func TestConnectionAddedSubscriberEnqueuesSyncJob(t *testing.T) {
	queue := &fakeTaskQueue{}
	subscriber := eventsV1.NewConnectionAddedSubscriber(
		inboxCommands.NewOutboxSyncAccountJobDispatcher(queue),
		nil,
	)

	detail, err := contractevents.MarshalConnectionAdded(contractevents.ConnectionAdded{
		EventID:      "evt-1",
		TenantSlug:   "acme",
		ConnectionID: "conn-1",
		Provider:     "gmail",
	})
	require.NoError(t, err)

	ctx := tenant.WithTenantID(context.Background(), "acme")
	require.NoError(t, subscriber.Handle(ctx, events.IntegrationEvent{
		DetailType: contractevents.ConnectionAddedDetailType,
		Detail:     detail,
	}))
	require.Len(t, queue.jobs, 1)
}
