package ports

import (
	"context"

	"github.com/bowerbird/internal/organization/domain"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	UpdateStatus(ctx context.Context, organizationID, status string) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	GetByID(ctx context.Context, id, userID string) (*domain.Organization, error)
	ListAll(ctx context.Context) ([]domain.Organization, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
