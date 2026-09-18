package tenant

import (
	"context"
	"errors"
)

type contextKey string

const tenantIdKey contextKey = "tenant_id"

// ErrNoTenantIdInContext is returned when no tenant id is found in the context.
var ErrNoTenantIdInContext = errors.New("no tenant id found in context")

// WithTenantID adds a tenant id to the context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIdKey, tenantID)
}

// TenantIDFromContext extracts the tenant id from the context.
func TenantIDFromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantIdKey).(string)
	if !ok || tenantID == "" {
		return "", ErrNoTenantIdInContext
	}

	return tenantID, nil
}
