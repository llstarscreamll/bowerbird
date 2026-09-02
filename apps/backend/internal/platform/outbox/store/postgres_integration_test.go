package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPostgresStoreClaimAndMarkProcessed(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `DELETE FROM outbox_events WHERE id = 'test-outbox-1'`)
	require.NoError(t, err)

	s := store.NewPostgresStore(pool)
	require.NoError(t, s.InsertEventStandalone(ctx, store.InsertEventInput{
		ID: "test-outbox-1", TenantSlug: "acme", Source: "bowerbird.test",
		DetailType: "TestEvent", Payload: []byte(`{}`), MaxAttempts: 3,
	}))

	rows, err := s.ClaimPendingEvents(ctx, 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, rows[0].Attempts)

	require.NoError(t, s.MarkEventProcessed(ctx, "test-outbox-1"))

	var status string
	var processedAt *time.Time
	err = pool.QueryRow(ctx, `SELECT status, processed_at FROM outbox_events WHERE id = 'test-outbox-1'`).Scan(&status, &processedAt)
	require.NoError(t, err)
	require.Equal(t, store.StatusProcessed, status)
	require.NotNil(t, processedAt)
}

func TestPostgresStoreMarkFailedAfterMaxAttempts(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `DELETE FROM outbox_jobs WHERE id = 'test-job-1'`)
	require.NoError(t, err)

	s := store.NewPostgresStore(pool)
	require.NoError(t, s.InsertJobStandalone(ctx, store.InsertJobInput{
		ID: "test-job-1", TenantSlug: "acme", JobType: "TestJob", Payload: []byte(`{}`), MaxAttempts: 2,
	}))

	require.NoError(t, s.MarkJobFailed(ctx, "test-job-1", "broker down", 2, 2))

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM outbox_jobs WHERE id = 'test-job-1'`).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, store.StatusFailed, status)
}

func TestPostgresStorePurgeTerminal(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	s := store.NewPostgresStore(pool)
	oldID := "purge-old-event"
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE id = $1`, oldID)

	require.NoError(t, s.InsertEventStandalone(ctx, store.InsertEventInput{
		ID: oldID, TenantSlug: "acme", Source: "test", DetailType: "T", Payload: []byte(`{}`),
	}))
	require.NoError(t, s.MarkEventProcessed(ctx, oldID))
	_, err = pool.Exec(ctx, `UPDATE outbox_events SET processed_at = NOW() - INTERVAL '8 days' WHERE id = $1`, oldID)
	require.NoError(t, err)

	events, jobs, err := s.PurgeTerminal(ctx, time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), events)
	require.Equal(t, int64(0), jobs)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE id = $1`, oldID).Scan(&count))
	require.Equal(t, 0, count)
}
