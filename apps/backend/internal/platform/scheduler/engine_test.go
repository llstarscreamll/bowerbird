package scheduler

import (
	"context"
	"testing"

	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTransport struct {
	jobs []store.JobRow
	err  error
}

func (f *fakeTransport) DeliverEvent(context.Context, store.EventRow) error { return nil }

func (f *fakeTransport) DeliverJob(_ context.Context, row store.JobRow) error {
	f.jobs = append(f.jobs, row)
	return f.err
}

func TestEngineDeliversOneTenantlessJob(t *testing.T) {
	transport := &fakeTransport{}
	engine, err := NewEngine(transport, []Rule{{
		Name:     "outbox-sweeper",
		Schedule: "rate(1 hour)",
		JobType:  OutboxSweeperJobType,
	}})
	require.NoError(t, err)

	engine.fire(context.Background(), engine.rules[0].Rule)

	require.Len(t, transport.jobs, 1)
	assert.Empty(t, transport.jobs[0].TenantSlug)
	assert.Equal(t, OutboxSweeperJobType, transport.jobs[0].JobType)
	assert.Equal(t, []byte(`{}`), transport.jobs[0].Payload)
	assert.NotEmpty(t, transport.jobs[0].ID)
	assert.Equal(t, transport.jobs[0].ID, transport.jobs[0].CorrelationID)
}

func TestNewEngineRejectsInvalidCatalog(t *testing.T) {
	transport := &fakeTransport{}

	_, err := NewEngine(nil, PlatformRules())
	require.Error(t, err)

	_, err = NewEngine(transport, nil)
	require.Error(t, err)

	_, err = NewEngine(transport, []Rule{
		{Name: "dup", Schedule: "rate(1 hour)", JobType: "A"},
		{Name: "dup", Schedule: "rate(1 hour)", JobType: "B"},
	})
	require.Error(t, err)

	_, err = NewEngine(transport, []Rule{
		{Name: "bad", Schedule: "rate(1 hours)", JobType: "A"},
	})
	require.Error(t, err)
}
