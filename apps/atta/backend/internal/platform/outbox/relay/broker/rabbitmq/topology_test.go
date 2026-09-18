package rabbitmq_test

import (
	"testing"
	"time"

	rabbitmq "github.com/atta/internal/platform/outbox/relay/broker/rabbitmq"
)

func TestTopologyConstants(t *testing.T) {
	requireEqual(t, rabbitmq.EventsExchange, "atta.events")
	requireEqual(t, rabbitmq.JobsExchange, "atta.jobs")
	requireEqual(t, rabbitmq.DLXExchange, "atta.dlx")
	requireEqual(t, rabbitmq.EventsQueue, "atta.events.handlers")
	requireEqual(t, rabbitmq.JobsQueue, "atta.jobs.work")
	requireEqual(t, rabbitmq.DeadLetterQueue, "atta.deadletter")
}

func TestConnectionBackoffDoubles(t *testing.T) {
	conn := rabbitmq.NewConnection("amqp://guest:guest@localhost:5672/")
	start := time.Millisecond * 100
	doubled := conn.BackoffForTest(start)
	if doubled != start*2 {
		t.Fatalf("expected doubled backoff")
	}
}

func requireEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
