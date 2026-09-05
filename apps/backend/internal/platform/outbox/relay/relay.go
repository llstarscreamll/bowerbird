package relay

import (
	"context"
	"log"
	"time"

	"github.com/bowerbird/internal/platform/outbox/relay/broker"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/tenant"
)

type Config struct {
	BatchSize    int
	PerTenantCap int
	PollInterval time.Duration
}

type Relay struct {
	store     store.RelayRepository
	transport broker.Transport
	cfg       Config
}

func New(store store.RelayRepository, transport broker.Transport, cfg Config) *Relay {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	return &Relay{store: store, transport: transport, cfg: cfg}
}

func (r *Relay) RunOnce(ctx context.Context) error {
	eventsDelivered, eventsFailed, err := r.processEvents(ctx)
	if err != nil {
		return err
	}
	jobsDelivered, jobsFailed, err := r.processJobs(ctx)
	if err != nil {
		return err
	}

	pendingEvents, pendingJobs, err := r.store.CountPending(ctx)
	if err != nil {
		return err
	}
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}
	log.Printf(
		"outbox relay tick: tenant=%s delivered_events=%d failed_events=%d delivered_jobs=%d failed_jobs=%d pending_events=%d pending_jobs=%d",
		tenantID, eventsDelivered, eventsFailed, jobsDelivered, jobsFailed, pendingEvents, pendingJobs,
	)
	return nil
}

func (r *Relay) RunLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := r.RunOnce(ctx); err != nil {
			log.Printf("outbox relay error: %v", err)
		}
		time.Sleep(r.cfg.PollInterval)
	}
}

func (r *Relay) processEvents(ctx context.Context) (delivered, failed int, err error) {
	rows, err := r.store.ClaimPendingEvents(ctx, r.cfg.BatchSize)
	if err != nil {
		return 0, 0, err
	}

	tenantCounts := make(map[string]int)
	for _, row := range rows {
		if r.cfg.PerTenantCap > 0 {
			if tenantCounts[row.TenantSlug] >= r.cfg.PerTenantCap {
				continue
			}
			tenantCounts[row.TenantSlug]++
		}

		if err := r.transport.DeliverEvent(ctx, row); err != nil {
			failed++
			if markErr := r.store.MarkEventFailed(ctx, row.ID, err.Error(), row.Attempts, row.MaxAttempts); markErr != nil {
				log.Printf("mark event failed: %v", markErr)
			}
			continue
		}
		if err := r.store.MarkEventProcessed(ctx, row.ID); err != nil {
			return delivered, failed, err
		}
		delivered++
	}
	return delivered, failed, nil
}

func (r *Relay) processJobs(ctx context.Context) (delivered, failed int, err error) {
	rows, err := r.store.ClaimPendingJobs(ctx, r.cfg.BatchSize)
	if err != nil {
		return 0, 0, err
	}

	tenantCounts := make(map[string]int)
	for _, row := range rows {
		if r.cfg.PerTenantCap > 0 {
			if tenantCounts[row.TenantSlug] >= r.cfg.PerTenantCap {
				continue
			}
			tenantCounts[row.TenantSlug]++
		}

		if err := r.transport.DeliverJob(ctx, row); err != nil {
			failed++
			if markErr := r.store.MarkJobFailed(ctx, row.ID, err.Error(), row.Attempts, row.MaxAttempts); markErr != nil {
				log.Printf("mark job failed: %v", markErr)
			}
			continue
		}
		if err := r.store.MarkJobProcessed(ctx, row.ID); err != nil {
			return delivered, failed, err
		}
		delivered++
	}
	return delivered, failed, nil
}
