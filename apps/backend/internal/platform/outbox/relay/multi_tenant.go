package relay

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/outbox/relay/broker"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/tenant"
	tenantapi "github.com/bowerbird/internal/tenant/api"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantLister returns slugs of active tenants.
type TenantLister interface {
	ListActiveTenantSlugs(ctx context.Context) ([]string, error)
}

// ControlPlaneTenantLister lists active tenants from the control-plane database.
type ControlPlaneTenantLister struct {
	pool *pgxpool.Pool
}

func NewControlPlaneTenantLister(pool *pgxpool.Pool) *ControlPlaneTenantLister {
	if pool == nil {
		panic("control plane pool is required")
	}
	return &ControlPlaneTenantLister{pool: pool}
}

func (l *ControlPlaneTenantLister) ListActiveTenantSlugs(ctx context.Context) ([]string, error) {
	rows, err := l.pool.Query(ctx, `
		SELECT slug FROM tenants WHERE status = $1 ORDER BY slug
	`, tenantapi.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("list active tenants: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

// MultiTenantRelay drains outbox tables for every active tenant (all deployment profiles).
type MultiTenantRelay struct {
	registry  *database.Registry
	lister    TenantLister
	transport broker.Transport
	cfg       Config
}

func NewMultiTenantRelay(registry *database.Registry, lister TenantLister, transport broker.Transport, cfg Config) *MultiTenantRelay {
	return &MultiTenantRelay{
		registry:  registry,
		lister:    lister,
		transport: transport,
		cfg:       cfg,
	}
}

func (m *MultiTenantRelay) RunOnce(ctx context.Context) error {
	slugs, err := m.lister.ListActiveTenantSlugs(ctx)
	if err != nil {
		return err
	}
	if len(slugs) == 0 {
		log.Printf("outbox relay: no active tenants to drain")
		return nil
	}

	for _, slug := range slugs {
		tenantCtx := tenant.WithTenantID(ctx, slug)
		pool, err := m.registry.GetPool(tenantCtx)
		if err != nil {
			return fmt.Errorf("tenant %s pool: %w", slug, err)
		}
		repo := store.NewPostgresStore(pool)
		r := New(repo, m.transport, m.cfg)
		if err := r.RunOnce(tenantCtx); err != nil {
			return fmt.Errorf("tenant %s relay: %w", slug, err)
		}
	}
	return nil
}

func (m *MultiTenantRelay) RunLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := m.RunOnce(ctx); err != nil {
			log.Printf("outbox relay error: %v", err)
		}
		sleep(ctx, m.cfg.PollInterval)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	if d <= 0 {
		d = time.Second
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
