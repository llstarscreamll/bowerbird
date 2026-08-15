package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/entitlements/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ResolveTenantID(ctx context.Context, idOrSlug string) (string, error) {
	var tenantID string
	err := r.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE id = $1 OR slug = $1 LIMIT 1`, idOrSlug).Scan(&tenantID)
	if err != nil {
		return "", fmt.Errorf("resolve tenant: %w", err)
	}
	return tenantID, nil
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID string) ([]domain.Entitlement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, feature_key, status, source, starts_at, ends_at, COALESCE(created_by, ''), created_at, updated_at
		FROM tenant_entitlements
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list entitlements: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Entitlement, 0)
	for rows.Next() {
		var item domain.Entitlement
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.FeatureKey,
			&item.Status,
			&item.Source,
			&item.StartsAt,
			&item.EndsAt,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan entitlement: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Upsert(ctx context.Context, entitlement domain.Entitlement) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenant_entitlements (id, tenant_id, feature_key, status, source, starts_at, ends_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10)
		ON CONFLICT (tenant_id, feature_key) DO UPDATE SET
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			starts_at = EXCLUDED.starts_at,
			ends_at = EXCLUDED.ends_at,
			updated_at = EXCLUDED.updated_at
	`, entitlement.ID, entitlement.TenantID, entitlement.FeatureKey, entitlement.Status, entitlement.Source, entitlement.StartsAt, entitlement.EndsAt, entitlement.CreatedBy, entitlement.CreatedAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("upsert entitlement: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, featureKey string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tenant_entitlements WHERE tenant_id = $1 AND feature_key = $2`, tenantID, featureKey)
	if err != nil {
		return fmt.Errorf("delete entitlement: %w", err)
	}
	return nil
}

func (r *Repository) ReplaceTenantFeatures(ctx context.Context, tenantID string, entitlements []domain.Entitlement) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM tenant_entitlements WHERE tenant_id = $1`, tenantID); err != nil {
		return err
	}
	for _, entitlement := range entitlements {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tenant_entitlements (id, tenant_id, feature_key, status, source, starts_at, ends_at, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10)
		`, entitlement.ID, entitlement.TenantID, entitlement.FeatureKey, entitlement.Status, entitlement.Source, entitlement.StartsAt, entitlement.EndsAt, entitlement.CreatedBy, entitlement.CreatedAt, entitlement.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
