package events

import (
	"testing"

	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func TestParseReceiveCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want int32
	}{
		{name: "empty defaults to one", raw: "", want: 1},
		{name: "invalid defaults to one", raw: "x", want: 1},
		{name: "zero defaults to one", raw: "0", want: 1},
		{name: "positive count", raw: "3", want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := parseReceiveCount(tc.raw); got != tc.want {
				t.Fatalf("parseReceiveCount(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}

func TestPollerBackoffVisibilityTimeout(t *testing.T) {
	t.Parallel()

	p := Poller{failureBackoffBaseSec: 5, failureBackoffMaxSec: 300}

	tests := []struct {
		name         string
		receiveCount int32
		want         int32
	}{
		{name: "first failure uses base", receiveCount: 1, want: 5},
		{name: "second failure doubles", receiveCount: 2, want: 10},
		{name: "third failure doubles again", receiveCount: 3, want: 20},
		{name: "caps at max", receiveCount: 10, want: 300},
		{name: "non-positive receives base", receiveCount: 0, want: 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := p.backoffVisibilityTimeout(tc.receiveCount); got != tc.want {
				t.Fatalf("backoffVisibilityTimeout(%d) = %d, want %d", tc.receiveCount, got, tc.want)
			}
		})
	}
}

func TestToSQSRecordsCopiesSystemAttributes(t *testing.T) {
	t.Parallel()

	messageID := "msg-1"
	receipt := "rh-1"
	body := "{}"
	message := sqstypes.Message{
		MessageId:     &messageID,
		ReceiptHandle: &receipt,
		Body:          &body,
		Attributes: map[string]string{
			"ApproximateReceiveCount": "4",
		},
		MessageAttributes: map[string]sqstypes.MessageAttributeValue{},
	}

	records := toSQSRecords([]sqstypes.Message{message})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if got := records[0].Attributes["ApproximateReceiveCount"]; got != "4" {
		t.Fatalf("record missing ApproximateReceiveCount, got %q", got)
	}
}
