package sweeper

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/scheduler"
	"github.com/bowerbird/internal/platform/tenant"
)

const defaultOutboxRetention = 7 * 24 * time.Hour

type Handler struct {
	registry  *database.Registry
	tenants   tenantLister
	retention time.Duration
}

type tenantLister interface {
	ListActiveTenantSlugs(ctx context.Context) ([]string, error)
}

func NewHandler(registry *database.Registry, tenants tenantLister, retention time.Duration) *Handler {
	if registry == nil {
		panic("tenant registry is required")
	}
	if tenants == nil {
		panic("tenant lister is required")
	}
	if retention <= 0 {
		retention = defaultOutboxRetention
	}
	return &Handler{registry: registry, tenants: tenants, retention: retention}
}

func (h *Handler) JobType() string {
	return scheduler.OutboxSweeperJobType
}

func (h *Handler) Scope() jobs.Scope { return jobs.ScopePlatform }

func (h *Handler) Handle(ctx context.Context, msg jobs.JobMessage) error {
	slugs, err := h.tenants.ListActiveTenantSlugs(ctx)
	if err != nil {
		return err
	}
	if len(slugs) == 0 {
		log.Printf("outbox sweeper: no active tenants job=%s", msg.MessageID)
		return nil
	}

	before := time.Now().Add(-h.retention)
	var purgeErr error
	ok := 0
	for _, slug := range slugs {
		if err := h.purgeTenant(ctx, slug, before); err != nil {
			log.Printf("outbox sweeper tenant=%s: %v", slug, err)
			purgeErr = errors.Join(purgeErr, err)
			continue
		}
		ok++
	}
	if ok > 0 {
		return nil
	}
	return purgeErr
}

func (h *Handler) purgeTenant(ctx context.Context, slug string, before time.Time) error {
	tenantCtx := tenant.WithTenantID(ctx, slug)
	pool, err := h.registry.GetPool(tenantCtx)
	if err != nil {
		return err
	}
	eventsDeleted, jobsDeleted, err := store.NewPostgresStore(pool).PurgeTerminal(ctx, before)
	if err != nil {
		return err
	}
	if eventsDeleted > 0 || jobsDeleted > 0 {
		log.Printf("outbox sweeper purged events=%d jobs=%d tenant=%s before=%s", eventsDeleted, jobsDeleted, slug, before.Format(time.RFC3339))
	}
	return nil
}
