package events

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type EventHandler struct{}

func NewEventHandler() EventHandler {
	return EventHandler{}
}

func (h EventHandler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		msgCtx := ctx
		// Extract Tenant ID from Message Attributes
		if attr, ok := record.MessageAttributes["TenantID"]; ok && attr.StringValue != nil {
			msgCtx = tenant.WithTenant(msgCtx, *attr.StringValue)
		}

		tenantID, _ := tenant.FromContext(msgCtx)
		log.Printf("sqs message processed: id=%s tenant=%s body=%s", record.MessageId, tenantID, record.Body)
	}

	return nil
}

func (h EventHandler) HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error {
	log.Printf("eventbridge event processed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
	return nil
}
