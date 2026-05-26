package domain

import "context"

type Repository interface {
	Create(ctx context.Context, org *Organization) error
	UpdateStatus(ctx context.Context, organizationID, status string) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	GetByID(ctx context.Context, id, userID string) (*Organization, error)
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
