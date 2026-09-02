package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumerRetryDelaysSpanMultipleDays(t *testing.T) {
	total := time.Duration(0)
	for _, delay := range ConsumerRetryDelays {
		total += delay
	}
	if total < 7*24*time.Hour {
		t.Fatalf("expected cumulative retry delay >= 7 days, got %v", total)
	}
}

func TestRetryQueueName(t *testing.T) {
	got := retryQueueName(JobsQueue, 2)
	want := "bowerbird.jobs.work.retry.tier2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsPermanentConsumerError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "internal", err: appErrors.New(appErrors.CodeInternal, "db down"), want: false},
		{name: "validation", err: appErrors.New(appErrors.CodeValidation, "bad payload"), want: true},
		{name: "attestation", err: attestation.ErrInvalidAttestation, want: true},
		{name: "json syntax", err: &json.SyntaxError{Offset: 1}, want: true},
		{name: "cancelled", err: context.Canceled, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsPermanentConsumerError(tc.err); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestConsumerRetryCount(t *testing.T) {
	if got := consumerRetryCount(nil); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
	if got := consumerRetryCount(amqp.Table{RetryCountHeader: int32(3)}); got != 3 {
		t.Fatalf("got %d want 3", got)
	}
}

func TestIsPermanentConsumerErrorWrapsAttestation(t *testing.T) {
	err := errors.Join(attestation.ErrMissingAttestation)
	if !IsPermanentConsumerError(err) {
		t.Fatal("expected wrapped attestation error to be permanent")
	}
}
