package publisher

import (
	"context"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/tenant"
)

type OutboxTaskQueue struct {
	store store.Appender
}

func NewOutboxTaskQueue(outboxStore store.Appender) *OutboxTaskQueue {
	if outboxStore == nil {
		panic("outbox store is required")
	}
	return &OutboxTaskQueue{store: outboxStore}
}

func (q *OutboxTaskQueue) Enqueue(ctx context.Context, job jobs.Job) error {
	tenantSlug, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	input := store.InsertJobInput{
		ID:            id.NewULID(),
		TenantSlug:    tenantSlug,
		JobType:       job.Type,
		Payload:       job.Payload,
		CorrelationID: correlationIDFromContext(ctx),
	}

	if tx, ok := database.TxFromContext(ctx); ok {
		return q.store.InsertJob(ctx, tx, input)
	}
	return q.store.InsertJobStandalone(ctx, input)
}
