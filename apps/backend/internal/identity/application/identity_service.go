package application

import (
	"context"
	"fmt"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/domain"
)

type IdentityService struct {
	repo domain.Repository
}

func NewIdentityService(repo domain.Repository) *IdentityService {
	return &IdentityService{repo: repo}
}

type TenantMembershipDTO struct {
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

func (s *IdentityService) ListUserTenants(ctx context.Context, userID string) ([]TenantMembershipDTO, error) {
	memberships, err := s.repo.FindTenantMemberships(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}

	dtos := make([]TenantMembershipDTO, len(memberships))
	for i, m := range memberships {
		dtos[i] = TenantMembershipDTO{
			TenantID: m.TenantID,
			Role:     string(m.Role),
		}
	}
	return dtos, nil
}

func (s *IdentityService) LeaveTenant(ctx context.Context, userID, tenantID string) error {
	// Remove from control plane (soft delete)
	err := s.repo.RemoveTenantMembership(ctx, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to leave tenant: %w", err)
	}

	// Update tenant DB
	dbName, err := s.repo.GetTenantDBName(ctx, tenantID)
	if err == nil && dbName != "" {
		_ = s.repo.SoftDeleteTenantUserProfile(ctx, dbName, userID)
	}

	return nil
}

func (s *IdentityService) DeleteAccount(ctx context.Context, userID string) error {
	// Find all tenants user belongs to, so we can soft delete them from tenant DBs too
	memberships, _ := s.repo.FindTenantMemberships(ctx, userID)

	// Soft delete from control plane
	err := s.repo.SoftDeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Obfuscate in tenant DBs
	for _, m := range memberships {
		dbName, err := s.repo.GetTenantDBName(ctx, m.TenantID)
		if err == nil && dbName != "" {
			_ = s.repo.SoftDeleteTenantUserProfile(ctx, dbName, userID)
		}
	}

	return nil
}
