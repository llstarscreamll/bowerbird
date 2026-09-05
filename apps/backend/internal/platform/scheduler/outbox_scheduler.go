package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
)

const (
	OutboxSweeperJobType = "platform.OutboxSweeper"
)

type TaskEnqueuer interface {
	Enqueue(ctx context.Context, job jobs.Job) error
}

type TenantLister interface {
	ListActiveTenantSlugs(ctx context.Context) ([]string, error)
}

type OutboxScheduler struct {
	queue    TaskEnqueuer
	tenants  TenantLister
	interval time.Duration
}

func NewOutboxScheduler(queue TaskEnqueuer, tenants TenantLister, interval time.Duration) *OutboxScheduler {
	if queue == nil {
		panic("task queue is required")
	}
	if tenants == nil {
		panic("tenant lister is required")
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &OutboxScheduler{queue: queue, tenants: tenants, interval: interval}
}

func (s *OutboxScheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *OutboxScheduler) tick(ctx context.Context) {
	slugs, err := s.tenants.ListActiveTenantSlugs(ctx)
	if err != nil {
		log.Printf("outbox scheduler: list tenants: %v", err)
		return
	}
	if len(slugs) == 0 {
		log.Printf("outbox scheduler: no active tenants")
		return
	}

	job := jobs.Job{Type: OutboxSweeperJobType, Payload: []byte(`{}`)}
	for _, slug := range slugs {
		tenantCtx := tenant.WithTenantID(ctx, slug)
		if err := s.queue.Enqueue(tenantCtx, job); err != nil {
			log.Printf("outbox scheduler: enqueue sweeper tenant=%s err=%v", slug, err)
		}
	}
}

func (s *OutboxScheduler) Stop(context.Context) error { return nil }
