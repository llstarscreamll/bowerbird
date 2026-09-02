package relay_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bowerbird/internal/platform/outbox/relay"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	events []store.EventRow
	jobs   []store.JobRow
	mu     sync.Mutex
}

func (m *mockStore) InsertEvent(context.Context, pgx.Tx, store.InsertEventInput) error   { return nil }
func (m *mockStore) InsertJob(context.Context, pgx.Tx, store.InsertJobInput) error       { return nil }
func (m *mockStore) InsertEventStandalone(context.Context, store.InsertEventInput) error { return nil }
func (m *mockStore) InsertJobStandalone(context.Context, store.InsertJobInput) error     { return nil }

func (m *mockStore) ClaimPendingEvents(_ context.Context, limit int) ([]store.EventRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || len(m.events) == 0 {
		return nil, nil
	}
	if limit > len(m.events) {
		limit = len(m.events)
	}
	out := append([]store.EventRow(nil), m.events[:limit]...)
	m.events = m.events[limit:]
	return out, nil
}

func (m *mockStore) ClaimPendingJobs(_ context.Context, limit int) ([]store.JobRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || len(m.jobs) == 0 {
		return nil, nil
	}
	if limit > len(m.jobs) {
		limit = len(m.jobs)
	}
	out := append([]store.JobRow(nil), m.jobs[:limit]...)
	m.jobs = m.jobs[limit:]
	return out, nil
}

func (m *mockStore) MarkEventProcessed(_ context.Context, _ string) error { return nil }
func (m *mockStore) MarkJobProcessed(_ context.Context, _ string) error   { return nil }
func (m *mockStore) MarkEventFailed(_ context.Context, _ string, _ string, _ int, _ int) error {
	return nil
}
func (m *mockStore) MarkJobFailed(_ context.Context, _ string, _ string, _ int, _ int) error {
	return nil
}
func (m *mockStore) IncrementEventAttempt(context.Context, string, string) error { return nil }
func (m *mockStore) IncrementJobAttempt(context.Context, string, string) error   { return nil }
func (m *mockStore) CountPending(context.Context) (int64, int64, error)          { return 0, 0, nil }
func (m *mockStore) PurgeTerminal(context.Context, time.Time) (int64, int64, error) {
	return 0, 0, nil
}

type mockTransport struct {
	events []string
}

func (t *mockTransport) DeliverEvent(_ context.Context, row store.EventRow) error {
	t.events = append(t.events, row.TenantSlug)
	return nil
}

func (t *mockTransport) DeliverJob(context.Context, store.JobRow) error { return nil }

func TestRelayFairMultiTenantCap(t *testing.T) {
	st := &mockStore{
		events: []store.EventRow{
			{ID: "1", TenantSlug: "a", MaxAttempts: 3},
			{ID: "2", TenantSlug: "a", MaxAttempts: 3},
			{ID: "3", TenantSlug: "b", MaxAttempts: 3},
			{ID: "4", TenantSlug: "b", MaxAttempts: 3},
		},
	}
	tr := &mockTransport{}
	r := relay.New(st, tr, relay.Config{BatchSize: 4, PerTenantCap: 1})

	require.NoError(t, r.RunOnce(context.Background()))
	require.Equal(t, []string{"a", "b"}, tr.events)
}

type failingTransport struct{}

func (failingTransport) DeliverEvent(context.Context, store.EventRow) error {
	return errors.New("broker down")
}
func (failingTransport) DeliverJob(context.Context, store.JobRow) error { return nil }

type poisonStore struct {
	mockStore
	failed []string
}

func (p *poisonStore) MarkEventFailed(_ context.Context, id, _ string, attempts, max int) error {
	if attempts >= max {
		p.failed = append(p.failed, id)
	}
	return nil
}

func TestRelayPoisonPillMarksFailed(t *testing.T) {
	st := &poisonStore{mockStore: mockStore{
		events: []store.EventRow{{ID: "evt-1", TenantSlug: "a", Attempts: 3, MaxAttempts: 3}},
	}}
	r := relay.New(st, failingTransport{}, relay.Config{BatchSize: 1})
	require.NoError(t, r.RunOnce(context.Background()))
	require.Equal(t, []string{"evt-1"}, st.failed)
}
