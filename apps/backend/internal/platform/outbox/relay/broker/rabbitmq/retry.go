package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	amqp "github.com/rabbitmq/amqp091-go"
)

const RetryCountHeader = "x-retry-count"

// ConsumerRetryDelays are per-attempt backoff tiers before the next delivery.
// Cumulative wait spans multiple days before the final attempt is dead-lettered.
var ConsumerRetryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	1 * time.Hour,
	4 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
	48 * time.Hour,
	72 * time.Hour,
	48 * time.Hour,
}

func retryQueueName(workQueue string, tier int) string {
	return fmt.Sprintf("%s.retry.tier%d", workQueue, tier)
}

func declareRetryQueues(ch *amqp.Channel, workQueue string, delays []time.Duration) error {
	for tier, delay := range delays {
		args := amqp.Table{
			"x-message-ttl":             int32(delay / time.Millisecond),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": workQueue,
		}
		if _, err := ch.QueueDeclare(retryQueueName(workQueue, tier), true, false, false, false, args); err != nil {
			return err
		}
	}
	return nil
}

// IsPermanentConsumerError reports errors that should not be retried (invalid payload, auth, etc.).
func IsPermanentConsumerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, attestation.ErrMissingAttestation) || errors.Is(err, attestation.ErrInvalidAttestation) {
		return true
	}

	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case appErrors.CodeValidation,
			appErrors.CodeUnauthorized,
			appErrors.CodeForbidden,
			appErrors.CodeNotFound,
			appErrors.CodeNotImplemented:
			return true
		}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}

	return false
}

func consumerRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers[RetryCountHeader]
	if !ok {
		return 0
	}
	switch n := raw.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func cloneHeaders(headers amqp.Table) amqp.Table {
	cloned := amqp.Table{}
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

// HandleConsumerFailure acks successful retries, dead-letters permanent or exhausted messages,
// and schedules delayed retries via TTL retry queues (native RabbitMQ, no delayed-exchange plugin).
func HandleConsumerFailure(ch *amqp.Channel, workQueue string, msg amqp.Delivery, err error) error {
	if IsPermanentConsumerError(err) {
		return msg.Nack(false, false)
	}

	retryCount := consumerRetryCount(msg.Headers)
	if retryCount >= len(ConsumerRetryDelays) {
		return msg.Nack(false, false)
	}

	headers := cloneHeaders(msg.Headers)
	headers[RetryCountHeader] = retryCount + 1

	if err := ch.Publish("", retryQueueName(workQueue, retryCount), false, false, amqp.Publishing{
		Headers:         headers,
		ContentType:     msg.ContentType,
		ContentEncoding: msg.ContentEncoding,
		DeliveryMode:    amqp.Persistent,
		MessageId:       msg.MessageId,
		Timestamp:       msg.Timestamp,
		Body:            msg.Body,
	}); err != nil {
		return msg.Nack(false, true)
	}

	return msg.Ack(false)
}
