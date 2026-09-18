package domain

import "context"

type Repository interface {
	Create(ctx context.Context, org *Tenant) error
	UpdateStatus(ctx context.Context, tenantID, status string) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	GetByID(ctx context.Context, id, userID string) (*Tenant, error)
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
