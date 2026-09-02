package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	StatusPending   = "pending"
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)

type EventRow struct {
	ID            string
	TenantSlug    string
	Source        string
	DetailType    string
	Payload       []byte
	CorrelationID string
	Status        string
	Attempts      int
	MaxAttempts   int
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	LastError     *string
}

type JobRow struct {
	ID            string
	TenantSlug    string
	JobType       string
	Payload       []byte
	CorrelationID string
	Status        string
	Attempts      int
	MaxAttempts   int
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	LastError     *string
}

type InsertEventInput struct {
	ID            string
	TenantSlug    string
	Source        string
	DetailType    string
	Payload       []byte
	CorrelationID string
	MaxAttempts   int
}

type InsertJobInput struct {
	ID            string
	TenantSlug    string
	JobType       string
	Payload       []byte
	CorrelationID string
	MaxAttempts   int
}

// Appender is the application-side transactional outbox port (insert path).
// Rows MUST be written in the same DB transaction as domain state when tx is non-nil.
// This mirrors the "outbox table write" side of the transactional outbox pattern.
type Appender interface {
	InsertEvent(ctx context.Context, tx pgx.Tx, input InsertEventInput) error
	InsertJob(ctx context.Context, tx pgx.Tx, input InsertJobInput) error
	InsertEventStandalone(ctx context.Context, input InsertEventInput) error
	InsertJobStandalone(ctx context.Context, input InsertJobInput) error
}

// RelayRepository is the relay/dispatcher-side outbox port (claim + finalize path).
// Used only by the outbox relay worker to drain pending rows and mark outcomes.
type RelayRepository interface {
	ClaimPendingEvents(ctx context.Context, limit int) ([]EventRow, error)
	ClaimPendingJobs(ctx context.Context, limit int) ([]JobRow, error)
	MarkEventProcessed(ctx context.Context, id string) error
	MarkJobProcessed(ctx context.Context, id string) error
	MarkEventFailed(ctx context.Context, id string, errMsg string, attempts int, maxAttempts int) error
	MarkJobFailed(ctx context.Context, id string, errMsg string, attempts int, maxAttempts int) error
	IncrementEventAttempt(ctx context.Context, id string, errMsg string) error
	IncrementJobAttempt(ctx context.Context, id string, errMsg string) error
	CountPending(ctx context.Context) (events int64, jobs int64, err error)
	PurgeTerminal(ctx context.Context, before time.Time) (eventsDeleted, jobsDeleted int64, err error)
}

// Repository combines both outbox ports for a single-tenant database (e.g. PostgresStore).
type Repository interface {
	Appender
	RelayRepository
}
