package ports

import (
	"context"

	"github.com/bowerbird/internal/identity/domain"
)

type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	CreateUserIdentity(ctx context.Context, identity *domain.UserIdentity) error
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	FindUserIdentityByProvider(ctx context.Context, userID, provider string) (*domain.UserIdentity, error)
	FindUserByID(ctx context.Context, userID string) (*domain.User, error)
	FindTenantMemberships(ctx context.Context, userID string) ([]*domain.TenantMembership, error)
	RemoveTenantMembership(ctx context.Context, userID, tenantID string) error
	GetTenantDBName(ctx context.Context, tenantID string) (string, error)
	SoftDeleteTenantUserProfile(ctx context.Context, dbName, userID string) error
	SoftDeleteUser(ctx context.Context, userID string) error
}
