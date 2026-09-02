package publisher

import (
	"context"

	"github.com/bowerbird/internal/platform/database"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/tenant"
)

type OutboxEventPublisher struct {
	store store.Appender
}

func NewOutboxEventPublisher(outboxStore store.Appender) *OutboxEventPublisher {
	if outboxStore == nil {
		panic("outbox store is required")
	}
	return &OutboxEventPublisher{store: outboxStore}
}

func (p *OutboxEventPublisher) Publish(ctx context.Context, event platformEvents.BusinessEvent) error {
	tenantSlug, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	correlationID := correlationIDFromContext(ctx)
	input := store.InsertEventInput{
		ID:            id.NewULID(),
		TenantSlug:    tenantSlug,
		Source:        event.Source,
		DetailType:    event.DetailType,
		Payload:       event.Detail,
		CorrelationID: correlationID,
	}

	if tx, ok := database.TxFromContext(ctx); ok {
		return p.store.InsertEvent(ctx, tx, input)
	}
	return p.store.InsertEventStandalone(ctx, input)
}

type correlationContextKey struct{}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationContextKey{}, correlationID)
}

func correlationIDFromContext(ctx context.Context) string {
	cid, _ := ctx.Value(correlationContextKey{}).(string)
	return cid
}
