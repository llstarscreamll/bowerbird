package ports

import (
	"context"

	"github.com/bowerbird/internal/tenant/domain"
)

type TenantRepository interface {
	Create(ctx context.Context, org *domain.Tenant) error
	UpdateStatus(ctx context.Context, tenantID, status string) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	GetByID(ctx context.Context, id, userID string) (*domain.Tenant, error)
	ListAll(ctx context.Context) ([]domain.Tenant, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
