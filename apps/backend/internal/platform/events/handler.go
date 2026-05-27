package events

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type EventBridgeSubscriber interface {
	DetailType() string
	HandleEventBridge(ctx context.Context, event events.CloudWatchEvent) error
}

type EventHandler struct {
	eventBridgeSubscribers map[string]EventBridgeSubscriber
}

func NewEventHandler(subscribers ...interface{}) EventHandler {
	ebRoutes := make(map[string]EventBridgeSubscriber)

	for _, subscriber := range subscribers {
		if subscriber == nil {
			continue
		}

		if ebSub, ok := subscriber.(EventBridgeSubscriber); ok {
			ebRoutes[ebSub.DetailType()] = ebSub
		}
	}

	return EventHandler{
		eventBridgeSubscribers: ebRoutes,
	}
}

func (h EventHandler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		msgCtx := ctx
		// Extract tenant slug from message attributes.
		if attr, ok := record.MessageAttributes["TenantSlug"]; ok && attr.StringValue != nil {
			msgCtx = tenant.WithTenantSlug(msgCtx, *attr.StringValue)
		}

		tenantSlug, _ := tenant.TenantSlugFromContext(msgCtx)
		log.Printf("sqs message processed: id=%s tenant=%s body=%s", record.MessageId, tenantSlug, record.Body)
	}

	return nil
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
