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
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
