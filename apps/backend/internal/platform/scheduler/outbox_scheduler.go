package scheduler

import (
	"context"
	"time"

	"github.com/bowerbird/internal/platform/jobs"
)

const (
	OutboxSweeperJobType = "platform.OutboxSweeper"
)

type TaskEnqueuer interface {
	Enqueue(ctx context.Context, job jobs.Job) error
}

type OutboxScheduler struct {
	queue    TaskEnqueuer
	interval time.Duration
}

func NewOutboxScheduler(queue TaskEnqueuer, interval time.Duration) *OutboxScheduler {
	if interval <= 0 {
		interval = time.Hour
	}
	return &OutboxScheduler{queue: queue, interval: interval}
}

func (s *OutboxScheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = s.queue.Enqueue(ctx, jobs.Job{Type: OutboxSweeperJobType, Payload: []byte(`{}`)})
		}
	}
}

func (s *OutboxScheduler) Stop(context.Context) error { return nil }
