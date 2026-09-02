package store

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/jackc/pgx/v5"
)

type RegistryStore struct {
	registry *database.Registry
}

func NewRegistryStore(registry *database.Registry) *RegistryStore {
	if registry == nil {
		panic("registry is required")
	}
	return &RegistryStore{registry: registry}
}

func (s *RegistryStore) poolForTenant(ctx context.Context, tenantSlug string) (*PostgresStore, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantSlug)
	pool, err := s.registry.GetPool(tenantCtx)
	if err != nil {
		return nil, fmt.Errorf("tenant pool %s: %w", tenantSlug, err)
	}
	return NewPostgresStore(pool), nil
}

func (s *RegistryStore) InsertEvent(ctx context.Context, tx pgx.Tx, input InsertEventInput) error {
	if tx != nil {
		_, err := tx.Exec(ctx, `
			INSERT INTO outbox_events (id, tenant_slug, source, detail_type, payload, correlation_id, max_attempts)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, input.ID, input.TenantSlug, input.Source, input.DetailType, input.Payload, nullIfEmpty(input.CorrelationID), defaultMaxAttempts(input.MaxAttempts))
		return err
	}
	store, err := s.poolForTenant(ctx, input.TenantSlug)
	if err != nil {
		return err
	}
	return store.InsertEventStandalone(ctx, input)
}

func (s *RegistryStore) InsertJob(ctx context.Context, tx pgx.Tx, input InsertJobInput) error {
	if tx != nil {
		_, err := tx.Exec(ctx, `
			INSERT INTO outbox_jobs (id, tenant_slug, job_type, payload, correlation_id, max_attempts)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, input.ID, input.TenantSlug, input.JobType, input.Payload, nullIfEmpty(input.CorrelationID), defaultMaxAttempts(input.MaxAttempts))
		return err
	}
	store, err := s.poolForTenant(ctx, input.TenantSlug)
	if err != nil {
		return err
	}
	return store.InsertJobStandalone(ctx, input)
}

func (s *RegistryStore) InsertEventStandalone(ctx context.Context, input InsertEventInput) error {
	store, err := s.poolForTenant(ctx, input.TenantSlug)
	if err != nil {
		return err
	}
	return store.InsertEventStandalone(ctx, input)
}

func (s *RegistryStore) InsertJobStandalone(ctx context.Context, input InsertJobInput) error {
	store, err := s.poolForTenant(ctx, input.TenantSlug)
	if err != nil {
		return err
	}
	return store.InsertJobStandalone(ctx, input)
}

func defaultMaxAttempts(v int) int {
	if v <= 0 {
		return 10
	}
	return v
}
