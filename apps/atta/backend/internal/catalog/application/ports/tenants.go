package ports

import "context"

type ActiveTenantLister interface {
	ListActiveTenantSlugs(ctx context.Context) ([]string, error)
}
