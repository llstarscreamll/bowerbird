package events

import (
	"context"
	"time"

	contractEvents "github.com/atta/internal/contracts/events"
	platformEvents "github.com/atta/internal/platform/events"
	"github.com/atta/internal/platform/id"
	"github.com/atta/internal/platform/tenant"
)

type Publisher struct {
	eventBus platformEvents.EventBus
}

func NewPublisher(eventBus platformEvents.EventBus) *Publisher {
	if eventBus == nil {
		panic("event bus is required")
	}
	return &Publisher{eventBus: eventBus}
}

func (p *Publisher) PublishRegistered(ctx context.Context) error {
	tenantSlug, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}
	event := contractEvents.LegalEntityRegistered{
		EventID:    id.NewULID(),
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		TenantSlug: tenantSlug,
	}
	payload, err := contractEvents.MarshalLegalEntityRegistered(event)
	if err != nil {
		return err
	}
	return p.eventBus.Publish(ctx, platformEvents.BusinessEvent{
		Source:     contractEvents.LegalEntityRegisteredSource,
		DetailType: contractEvents.LegalEntityRegisteredDetailType,
		Detail:     payload,
	})
}
