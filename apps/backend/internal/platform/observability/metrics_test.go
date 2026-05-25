package observability

import (
	"testing"
	"time"
)

func TestInMemoryMetricsCounterAndDuration(t *testing.T) {
	m := NewInMemoryMetrics()
	tags := map[string]string{"tenant_slug": "t1", "status": "ok"}

	m.IncCounter("inbox_sync_messages_total", tags)
	m.IncCounter("inbox_sync_messages_total", tags)
	m.ObserveDuration("invoicing_processing_latency_ms", 25*time.Millisecond, tags)

	if got := m.CounterValue("inbox_sync_messages_total", tags); got != 2 {
		t.Fatalf("expected counter 2, got %d", got)
	}

	if got := m.DurationCount("invoicing_processing_latency_ms", tags); got != 1 {
		t.Fatalf("expected duration count 1, got %d", got)
	}
}
