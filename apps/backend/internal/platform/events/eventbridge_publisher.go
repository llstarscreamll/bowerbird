package events

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/money-path/bowerbird/apps/backend/internal/connections/domain"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type EventBridgePublisher struct {
	client   *eventbridge.Client
	eventBus string
}

func NewEventBridgePublisher(client *eventbridge.Client, eventBus string) *EventBridgePublisher {
	return &EventBridgePublisher{
		client:   client,
		eventBus: eventBus,
	}
}

func (p *EventBridgePublisher) PublishConnectionAdded(ctx context.Context, connection *domain.Connection) error {
	tenantSlug, _ := tenant.TenantSlugFromContext(ctx)

	event := contractevents.ConnectionAdded{
		EventID:              id.NewULID(),
		OccurredAt:           time.Now().UTC().Format(time.RFC3339Nano),
		TenantSlug:           tenantSlug,
		ConnectionID:         connection.ID,
		Provider:             connection.Provider,
		ProviderAccountEmail: connection.ProviderAccountEmail,
	}

	payload, err := contractevents.MarshalConnectionAdded(event)
	if err != nil {
		return err
	}

	_, err = p.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(p.eventBus),
				Source:       aws.String(contractevents.ConnectionAddedSource),
				DetailType:   aws.String(contractevents.ConnectionAddedDetailType),
				Detail:       aws.String(string(payload)),
			},
		},
	})
	return err
}
