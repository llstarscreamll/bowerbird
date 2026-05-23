package handlers

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

type EventHandler struct{}

func NewEventHandler() EventHandler {
	return EventHandler{}
}

func (h EventHandler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		log.Printf("sqs message processed: id=%s body=%s", record.MessageId, record.Body)
	}

	return nil
}

func (h EventHandler) HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error {
	log.Printf("eventbridge event processed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
	return nil
}
