package domain

import (
	"context"
)

type Repository interface {
	// Identity Operations (Control Plane)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, user *User) error

	FindUserIdentity(ctx context.Context, provider, providerID string) (*UserIdentity, error)
	FindUserIdentityByProvider(ctx context.Context, userID, provider string) (*UserIdentity, error)
	CreateUserIdentity(ctx context.Context, identity *UserIdentity) error

	// Tenant Membership (Control Plane)
	FindTenantMemberships(ctx context.Context, userID string) ([]*TenantMembership, error)
	AddTenantMembership(ctx context.Context, membership *TenantMembership) error
	RemoveTenantMembership(ctx context.Context, userID, tenantID string) error

	// Tenant Profile Operations (Tenant DB)
	// Requires standard injection of the tenant's pgxpool
	CreateTenantUserProfile(ctx context.Context, tenantDBName string, profile *TenantUserProfile) error
	UpdateTenantUserProfile(ctx context.Context, tenantDBName string, profile *TenantUserProfile) error
	SoftDeleteTenantUserProfile(ctx context.Context, tenantDBName string, userID string) error

	// Administrative
	SoftDeleteUser(ctx context.Context, userID string) error
	GetTenantDBName(ctx context.Context, tenantID string) (string, error)
}
