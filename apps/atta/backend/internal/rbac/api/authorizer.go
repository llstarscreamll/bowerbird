package api

import (
	"context"

	"github.com/atta/internal/rbac/domain"
)

const (
	PermissionSecretsRead   = domain.PermissionSecretsRead
	PermissionSecretsWrite  = domain.PermissionSecretsWrite
	PermissionSecretsDelete = domain.PermissionSecretsDelete
)

// Authorizer is the RBAC Open Host Service for permission checks.
type Authorizer interface {
	RequirePermission(ctx context.Context, code string) error
}
