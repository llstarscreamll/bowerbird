package domain

import "context"

type Repository interface {
	Create(ctx context.Context, org *Organization) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	AddMembership(ctx context.Context, userID, tenantID, role string) error
}
