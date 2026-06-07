package events

import (
	"context"
	"time"

	"github.com/bowerbird/internal/connections/domain"
	contractEvents "github.com/bowerbird/internal/contracts/events"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/tenant"
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

func (p *Publisher) PublishConnectionAdded(ctx context.Context, connection *domain.Connection) error {
	tenantSlug, _ := tenant.TenantIDFromContext(ctx)

	event := contractEvents.ConnectionAdded{
		EventID:              id.NewULID(),
		OccurredAt:           time.Now().UTC().Format(time.RFC3339Nano),
		TenantSlug:           tenantSlug,
		ConnectionID:         connection.ID,
		Provider:             connection.Provider,
		ProviderAccountEmail: connection.ProviderAccountEmail,
	}

	payload, err := contractEvents.MarshalConnectionAdded(event)
	if err != nil {
		return err
	}

	return p.eventBus.Publish(ctx, platformEvents.BusinessEvent{
		Source:     contractEvents.ConnectionAddedSource,
		DetailType: contractEvents.ConnectionAddedDetailType,
		Detail:     payload,
	})
}
