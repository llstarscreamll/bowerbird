package domain

import "context"

type Repository interface {
	ResolveTenantID(ctx context.Context, idOrSlug string) (string, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Entitlement, error)
	Upsert(ctx context.Context, entitlement Entitlement) error
	Delete(ctx context.Context, tenantID, featureKey string) error
	ReplaceTenantFeatures(ctx context.Context, tenantID string, entitlements []Entitlement) error
}
