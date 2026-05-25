package tenant

import (
	"context"
	"errors"
)

type contextKey string

const tenantSlugKey contextKey = "tenant_slug"

// ErrNoTenantSlugInContext is returned when no tenant slug is found in the context.
var ErrNoTenantSlugInContext = errors.New("no tenant slug found in context")

// WithTenantSlug adds a tenant slug to the context.
func WithTenantSlug(ctx context.Context, tenantSlug string) context.Context {
	return context.WithValue(ctx, tenantSlugKey, tenantSlug)
}

// TenantSlugFromContext extracts the tenant slug from the context.
func TenantSlugFromContext(ctx context.Context) (string, error) {
	tenantSlug, ok := ctx.Value(tenantSlugKey).(string)
	if !ok || tenantSlug == "" {
		return "", ErrNoTenantSlugInContext
	}
	return tenantSlug, nil
}
