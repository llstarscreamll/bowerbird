package events

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Handler interface {
	HandleSQSEvent(ctx context.Context, event events.SQSEvent) error
	HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error
}

type Poller struct {
	client               *sqs.Client
	handler              Handler
	sqsQueueURL          string
	eventBridgeQueueURL  string
	waitTimeSeconds      int32
	visibilityTimeoutSec int32
}

func NewPoller(client *sqs.Client, handler Handler, sqsQueueURL, eventBridgeQueueURL string) Poller {
	return Poller{
		client:               client,
		handler:              handler,
		sqsQueueURL:          sqsQueueURL,
		eventBridgeQueueURL:  eventBridgeQueueURL,
		waitTimeSeconds:      10,
		visibilityTimeoutSec: 30,
	}
}

func (p Poller) Run(ctx context.Context) {
	if p.sqsQueueURL != "" {
		go p.pollSQS(ctx, p.sqsQueueURL, p.handler.HandleSQSEvent)
	}

	if p.eventBridgeQueueURL != "" {
		go p.pollEventBridge(ctx, p.eventBridgeQueueURL, p.handler.HandleEventBridgeEvent)
	}
}

func (p Poller) pollSQS(ctx context.Context, queueURL string, handler func(context.Context, events.SQSEvent) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messages, err := p.receiveMessages(ctx, queueURL)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			log.Printf("sqs poll error (queue=%s): %v", queueURL, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		event := events.SQSEvent{Records: toSQSRecords(messages)}
		if err := handler(ctx, event); err != nil {
			log.Printf("sqs handler error (queue=%s): %v", queueURL, err)
			continue
		}

		if err := p.deleteMessages(ctx, queueURL, messages); err != nil {
			log.Printf("sqs delete error (queue=%s): %v", queueURL, err)
		}
	}
}

func (p Poller) pollEventBridge(ctx context.Context, queueURL string, handler func(context.Context, events.CloudWatchEvent) error) {
	p.pollSQS(ctx, queueURL, func(ctx context.Context, event events.SQSEvent) error {
		for _, record := range event.Records {
			var bridgeEvent events.CloudWatchEvent
			if err := json.Unmarshal([]byte(record.Body), &bridgeEvent); err != nil {
				return err
			}

			if err := handler(ctx, bridgeEvent); err != nil {
				return err
			}
		}

		return nil
	})
}

func (p Poller) receiveMessages(ctx context.Context, queueURL string) ([]types.Message, error) {
	output, err := p.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     p.waitTimeSeconds,
		VisibilityTimeout:   p.visibilityTimeoutSec,
	})
	if err != nil {
		return nil, err
	}

	return output.Messages, nil
}

func (p Poller) deleteMessages(ctx context.Context, queueURL string, messages []types.Message) error {
	entries := make([]types.DeleteMessageBatchRequestEntry, 0, len(messages))
	for _, message := range messages {
		if message.MessageId == nil || message.ReceiptHandle == nil {
			continue
		}

		entries = append(entries, types.DeleteMessageBatchRequestEntry{
			Id:            message.MessageId,
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	_, err := p.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		QueueUrl: &queueURL,
		Entries:  entries,
	})

	return err
}

func toSQSRecords(messages []types.Message) []events.SQSMessage {
	records := make([]events.SQSMessage, 0, len(messages))
	for _, message := range messages {
		record := events.SQSMessage{}
		if message.MessageId != nil {
			record.MessageId = *message.MessageId
		}
		if message.Body != nil {
			record.Body = *message.Body
		}
		record.ReceiptHandle = ""
		record.Attributes = map[string]string{}
		record.MessageAttributes = map[string]events.SQSMessageAttribute{}
		record.Md5OfBody = ""
		record.EventSource = "aws:sqs"
		record.EventSourceARN = ""
		record.AWSRegion = ""
		records = append(records, record)
	}

	return records
}
