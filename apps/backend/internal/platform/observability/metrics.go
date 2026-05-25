package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Metrics interface {
	IncCounter(name string, tags map[string]string)
	ObserveDuration(name string, duration time.Duration, tags map[string]string)
}

type NoopMetrics struct{}

func (NoopMetrics) IncCounter(name string, tags map[string]string) {}

func (NoopMetrics) ObserveDuration(name string, duration time.Duration, tags map[string]string) {}

type InMemoryMetrics struct {
	mu        sync.Mutex
	counters  map[string]int64
	durations map[string][]time.Duration
}

func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:  make(map[string]int64),
		durations: make(map[string][]time.Duration),
	}
}

func (m *InMemoryMetrics) IncCounter(name string, tags map[string]string) {
	key := metricKey(name, tags)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key]++
}

func (m *InMemoryMetrics) ObserveDuration(name string, duration time.Duration, tags map[string]string) {
	key := metricKey(name, tags)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations[key] = append(m.durations[key], duration)
}

func (m *InMemoryMetrics) CounterValue(name string, tags map[string]string) int64 {
	key := metricKey(name, tags)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[key]
}

func (m *InMemoryMetrics) DurationCount(name string, tags map[string]string) int {
	key := metricKey(name, tags)
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.durations[key])
}

func metricKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(tags)+1)
	parts = append(parts, name)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, tags[k]))
	}
	return strings.Join(parts, "|")
}
