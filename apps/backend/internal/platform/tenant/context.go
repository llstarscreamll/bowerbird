package tenant

import (
	"context"
	"errors"
)

type contextKey string

const tenantKey contextKey = "tenant_id"

// ErrNoTenantInContext is returned when no tenant ID is found in the context
var ErrNoTenantInContext = errors.New("no tenant id found in context")

// WithTenant adds a tenant ID to the context
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// FromContext extracts the tenant ID from the context
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantKey).(string)
	if !ok || tenantID == "" {
		return "", ErrNoTenantInContext
	}
	return tenantID, nil
}
