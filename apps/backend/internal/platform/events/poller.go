package events

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Handler interface {
	HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error
}

type Poller struct {
	client                *sqs.Client
	handler               Handler
	eventBridgeQueueURL   string
	waitTimeSeconds       int32
	visibilityTimeoutSec  int32
	failureBackoffBaseSec int32
	failureBackoffMaxSec  int32
}

func NewPoller(client *sqs.Client, handler Handler, eventBridgeQueueURL string) Poller {
	return Poller{
		client:                client,
		handler:               handler,
		eventBridgeQueueURL:   eventBridgeQueueURL,
		waitTimeSeconds:       10,
		visibilityTimeoutSec:  30,
		failureBackoffBaseSec: 5,
		failureBackoffMaxSec:  300,
	}
}

func (p Poller) Run(ctx context.Context) {
	if p.eventBridgeQueueURL != "" && p.handler != nil {
		go p.pollEventBridge(ctx, p.eventBridgeQueueURL, p.handler.HandleEventBridgeEvent)
	}
}

func (p Poller) pollEventBridge(ctx context.Context, queueURL string, handler func(context.Context, events.CloudWatchEvent) error) {
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

		if err := p.dispatchEvents(ctx, messages, handler); err != nil {
			log.Printf("sqs handler error (queue=%s): %v", queueURL, err)
			if backoffErr := p.applyFailureBackoff(ctx, queueURL, messages); backoffErr != nil {
				log.Printf("sqs backoff error (queue=%s): %v", queueURL, backoffErr)
			}
			continue
		}

		if err := p.deleteMessages(ctx, queueURL, messages); err != nil {
			log.Printf("sqs delete error (queue=%s): %v", queueURL, err)
		}
	}
}

func (p Poller) dispatchEvents(ctx context.Context, messages []types.Message, handler func(context.Context, events.CloudWatchEvent) error) error {
	for _, message := range messages {
		if message.Body == nil {
			continue
		}

		var bridgeEvent events.CloudWatchEvent
		if err := json.Unmarshal([]byte(*message.Body), &bridgeEvent); err != nil {
			return err
		}

		if err := handler(ctx, bridgeEvent); err != nil {
			return err
		}
	}

	return nil
}

func (p Poller) receiveMessages(ctx context.Context, queueURL string) ([]types.Message, error) {
	output, err := p.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              &queueURL,
		MaxNumberOfMessages:   10,
		WaitTimeSeconds:       p.waitTimeSeconds,
		VisibilityTimeout:     p.visibilityTimeoutSec,
		AttributeNames:        []types.QueueAttributeName{"All"},
		MessageAttributeNames: []string{"All"},
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

func (p Poller) applyFailureBackoff(ctx context.Context, queueURL string, messages []types.Message) error {
	entries := make([]types.ChangeMessageVisibilityBatchRequestEntry, 0, len(messages))
	for _, message := range messages {
		if message.MessageId == nil || message.ReceiptHandle == nil {
			continue
		}

		receiveCount := parseReceiveCount(message.Attributes["ApproximateReceiveCount"])
		visibility := p.backoffVisibilityTimeout(receiveCount)

		entries = append(entries, types.ChangeMessageVisibilityBatchRequestEntry{
			Id:                message.MessageId,
			ReceiptHandle:     message.ReceiptHandle,
			VisibilityTimeout: visibility,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	_, err := p.client.ChangeMessageVisibilityBatch(ctx, &sqs.ChangeMessageVisibilityBatchInput{
		QueueUrl: &queueURL,
		Entries:  entries,
	})
	return err
}

func (p Poller) backoffVisibilityTimeout(receiveCount int32) int32 {
	if receiveCount <= 1 {
		return p.failureBackoffBaseSec
	}

	visibility := p.failureBackoffBaseSec << (receiveCount - 1)
	if visibility > p.failureBackoffMaxSec {
		return p.failureBackoffMaxSec
	}

	return visibility
}

func parseReceiveCount(raw string) int32 {
	if raw == "" {
		return 1
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v <= 0 {
		return 1
	}
	return int32(v)
}
