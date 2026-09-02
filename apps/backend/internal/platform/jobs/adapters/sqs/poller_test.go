package sqs

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
	if got := p.backoffVisibilityTimeout(2); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func TestToSQSRecordsCopiesSystemAttributes(t *testing.T) {
	t.Parallel()
	messageID := "msg-1"
	body := "{}"
	message := sqstypes.Message{
		MessageId: &messageID,
		Body:      &body,
		Attributes: map[string]string{
			"ApproximateReceiveCount": "4",
		},
	}
	records := toSQSRecords([]sqstypes.Message{message})
	if records[0].Attributes["ApproximateReceiveCount"] != "4" {
		t.Fatal("missing attribute")
	}
}
