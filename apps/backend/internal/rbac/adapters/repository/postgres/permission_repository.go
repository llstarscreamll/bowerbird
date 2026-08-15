package postgres

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/rbac/application/ports"
)

type PermissionRepository struct {
	registry *database.Registry
}

func NewPermissionRepository(registry *database.Registry) *PermissionRepository {
	return &PermissionRepository{registry: registry}
}

var _ ports.PermissionRepository = (*PermissionRepository)(nil)

func (r *PermissionRepository) ListCodesForUser(ctx context.Context, userID string) ([]string, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT DISTINCT p.code
		FROM user_roles ur
		INNER JOIN role_permissions rp ON rp.role_id = ur.role_id
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.user_id = $1
		ORDER BY p.code
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	defer rows.Close()

	codes := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return codes, nil
}
