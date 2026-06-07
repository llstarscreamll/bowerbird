package events

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
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

func (p *EventBridgePublisher) Publish(ctx context.Context, event BusinessEvent) error {
	_, err := p.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(p.eventBus),
				Source:       aws.String(event.Source),
				DetailType:   aws.String(event.DetailType),
				Detail:       aws.String(string(event.Detail)),
			},
		},
	})
	return err
}
