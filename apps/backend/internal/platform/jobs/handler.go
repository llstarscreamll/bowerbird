package jobs

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bowerbird/internal/platform/tenant"
)

type SQSProcessor interface {
	JobType() string
	HandleSQS(ctx context.Context, message events.SQSMessage) error
}

type Handler struct {
	processors map[string]SQSProcessor
}

func NewHandler(processors ...SQSProcessor) Handler {
	routes := make(map[string]SQSProcessor)

	for _, processor := range processors {
		if processor == nil {
			continue
		}

		routes[processor.JobType()] = processor
	}

	return Handler{processors: routes}
}

func (h Handler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		msgCtx := ctx
		if attr, ok := record.MessageAttributes["TenantID"]; ok && attr.StringValue != nil {
			msgCtx = tenant.WithTenantID(msgCtx, *attr.StringValue)
		}

		if attr, ok := record.MessageAttributes["JobType"]; ok && attr.StringValue != nil {
			if processor, found := h.processors[*attr.StringValue]; found {
				if err := processor.HandleSQS(msgCtx, record); err != nil {
					return err
				}

				tenantID, _ := tenant.TenantIDFromContext(msgCtx)
				log.Printf("sqs job routed: id=%s type=%s tenant=%s", record.MessageId, *attr.StringValue, tenantID)
				continue
			}
		}

		tenantID, _ := tenant.TenantIDFromContext(msgCtx)
		log.Printf("sqs message processed without job processor: id=%s tenant=%s", record.MessageId, tenantID)
	}

	return nil
}
