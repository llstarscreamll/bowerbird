package ports

import "context"

type PermissionRepository interface {
	ListCodesForUser(ctx context.Context, userID string) ([]string, error)
}
