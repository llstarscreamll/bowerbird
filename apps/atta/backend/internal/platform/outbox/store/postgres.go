package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	if pool == nil {
		panic("pool is required")
	}
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) InsertEvent(ctx context.Context, tx pgx.Tx, input InsertEventInput) error {
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (id, tenant_slug, source, detail_type, payload, correlation_id, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, input.ID, input.TenantSlug, input.Source, input.DetailType, input.Payload, nullIfEmpty(input.CorrelationID), maxAttempts)
	return err
}

func (s *PostgresStore) InsertJob(ctx context.Context, tx pgx.Tx, input InsertJobInput) error {
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_jobs (id, tenant_slug, job_type, payload, correlation_id, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, input.ID, input.TenantSlug, input.JobType, input.Payload, nullIfEmpty(input.CorrelationID), maxAttempts)
	return err
}

func (s *PostgresStore) InsertEventStandalone(ctx context.Context, input InsertEventInput) error {
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO outbox_events (id, tenant_slug, source, detail_type, payload, correlation_id, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, input.ID, input.TenantSlug, input.Source, input.DetailType, input.Payload, nullIfEmpty(input.CorrelationID), maxAttempts)
	return err
}

func (s *PostgresStore) InsertJobStandalone(ctx context.Context, input InsertJobInput) error {
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO outbox_jobs (id, tenant_slug, job_type, payload, correlation_id, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, input.ID, input.TenantSlug, input.JobType, input.Payload, nullIfEmpty(input.CorrelationID), maxAttempts)
	return err
}

func (s *PostgresStore) ClaimPendingEvents(ctx context.Context, limit int) ([]EventRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id FROM outbox_events
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events e
		SET attempts = e.attempts + 1
		FROM claimed c
		WHERE e.id = c.id
		RETURNING e.id, e.tenant_slug, e.source, e.detail_type, e.payload, e.correlation_id,
		          e.status, e.attempts, e.max_attempts, e.created_at, e.processed_at, e.last_error
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEventRows(rows)
}

func (s *PostgresStore) ClaimPendingJobs(ctx context.Context, limit int) ([]JobRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id FROM outbox_jobs
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_jobs j
		SET attempts = j.attempts + 1
		FROM claimed c
		WHERE j.id = c.id
		RETURNING j.id, j.tenant_slug, j.job_type, j.payload, j.correlation_id,
		          j.status, j.attempts, j.max_attempts, j.created_at, j.processed_at, j.last_error
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanJobRows(rows)
}

func (s *PostgresStore) MarkEventProcessed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events SET status = 'processed', processed_at = NOW(), last_error = NULL WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) MarkJobProcessed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_jobs SET status = 'processed', processed_at = NOW(), last_error = NULL WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) MarkEventFailed(ctx context.Context, id string, errMsg string, attempts int, maxAttempts int) error {
	status := StatusPending
	if attempts >= maxAttempts {
		status = StatusFailed
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events SET status = $2, last_error = $3 WHERE id = $1
	`, id, status, errMsg)
	return err
}

func (s *PostgresStore) MarkJobFailed(ctx context.Context, id string, errMsg string, attempts int, maxAttempts int) error {
	status := StatusPending
	if attempts >= maxAttempts {
		status = StatusFailed
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_jobs SET status = $2, last_error = $3 WHERE id = $1
	`, id, status, errMsg)
	return err
}

func (s *PostgresStore) IncrementEventAttempt(ctx context.Context, id string, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events SET last_error = $2 WHERE id = $1
	`, id, errMsg)
	return err
}

func (s *PostgresStore) IncrementJobAttempt(ctx context.Context, id string, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_jobs SET last_error = $2 WHERE id = $1
	`, id, errMsg)
	return err
}

func (s *PostgresStore) PurgeTerminal(ctx context.Context, before time.Time) (int64, int64, error) {
	tagEvents, err := s.pool.Exec(ctx, `
		DELETE FROM outbox_events
		WHERE status IN ('processed', 'failed')
		  AND COALESCE(processed_at, created_at) < $1
	`, before)
	if err != nil {
		return 0, 0, fmt.Errorf("purge outbox_events: %w", err)
	}
	tagJobs, err := s.pool.Exec(ctx, `
		DELETE FROM outbox_jobs
		WHERE status IN ('processed', 'failed')
		  AND COALESCE(processed_at, created_at) < $1
	`, before)
	if err != nil {
		return 0, 0, fmt.Errorf("purge outbox_jobs: %w", err)
	}
	return tagEvents.RowsAffected(), tagJobs.RowsAffected(), nil
}

func (s *PostgresStore) CountPending(ctx context.Context) (int64, int64, error) {
	var events, jobs int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = 'pending'`).Scan(&events); err != nil {
		return 0, 0, fmt.Errorf("count pending events: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_jobs WHERE status = 'pending'`).Scan(&jobs); err != nil {
		return 0, 0, fmt.Errorf("count pending jobs: %w", err)
	}
	return events, jobs, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func scanEventRows(rows pgx.Rows) ([]EventRow, error) {
	var out []EventRow
	for rows.Next() {
		var row EventRow
		var correlationID, lastError *string
		if err := rows.Scan(
			&row.ID, &row.TenantSlug, &row.Source, &row.DetailType, &row.Payload,
			&correlationID, &row.Status, &row.Attempts, &row.MaxAttempts,
			&row.CreatedAt, &row.ProcessedAt, &lastError,
		); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}
		if correlationID != nil {
			row.CorrelationID = *correlationID
		}
		row.LastError = lastError
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanJobRows(rows pgx.Rows) ([]JobRow, error) {
	var out []JobRow
	for rows.Next() {
		var row JobRow
		var correlationID, lastError *string
		if err := rows.Scan(
			&row.ID, &row.TenantSlug, &row.JobType, &row.Payload,
			&correlationID, &row.Status, &row.Attempts, &row.MaxAttempts,
			&row.CreatedAt, &row.ProcessedAt, &lastError,
		); err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		if correlationID != nil {
			row.CorrelationID = *correlationID
		}
		row.LastError = lastError
		out = append(out, row)
	}
	return out, rows.Err()
}
