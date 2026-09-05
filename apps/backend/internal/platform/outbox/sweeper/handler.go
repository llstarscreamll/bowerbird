package sweeper

import (
	"context"
	"log"
	"strings"
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
	retention time.Duration
}

func NewHandler(registry *database.Registry, retention time.Duration) *Handler {
	if registry == nil {
		panic("tenant registry is required")
	}
	if retention <= 0 {
		retention = defaultOutboxRetention
	}
	return &Handler{registry: registry, retention: retention}
}

func (h *Handler) JobType() string {
	return scheduler.OutboxSweeperJobType
}

func (h *Handler) Handle(ctx context.Context, msg jobs.JobMessage) error {
	slug := strings.TrimSpace(msg.TenantSlug)
	if slug == "" {
		return tenant.ErrNoTenantIdInContext
	}
	tenantCtx := tenant.WithTenantID(ctx, slug)
	pool, err := h.registry.GetPool(tenantCtx)
	if err != nil {
		return err
	}
	purger := store.NewPostgresStore(pool)
	before := time.Now().Add(-h.retention)
	events, jobsDeleted, err := purger.PurgeTerminal(ctx, before)
	if err != nil {
		return err
	}
	if events > 0 || jobsDeleted > 0 {
		log.Printf("outbox sweeper purged events=%d jobs=%d tenant=%s before=%s", events, jobsDeleted, slug, before.Format(time.RFC3339))
	}
	return nil
}
