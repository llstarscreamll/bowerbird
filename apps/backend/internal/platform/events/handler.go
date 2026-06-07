package events

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

type EventBridgeSubscriber interface {
	DetailType() string
	HandleEventBridge(ctx context.Context, event events.CloudWatchEvent) error
}

type EventHandler struct {
	eventBridgeSubscribers map[string]EventBridgeSubscriber
}

func NewEventHandler(subscribers ...EventBridgeSubscriber) EventHandler {
	ebRoutes := make(map[string]EventBridgeSubscriber)

	for _, subscriber := range subscribers {
		if subscriber == nil {
			continue
		}

		ebRoutes[subscriber.DetailType()] = subscriber
	}

	return EventHandler{
		eventBridgeSubscribers: ebRoutes,
	}
}

func (h EventHandler) HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error {
	if subscriber, ok := h.eventBridgeSubscribers[event.DetailType]; ok {
		if err := subscriber.HandleEventBridge(ctx, event); err != nil {
			return err
		}

		log.Printf("eventbridge event routed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
		return nil
	}

	log.Printf("eventbridge event processed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
	return nil
}
