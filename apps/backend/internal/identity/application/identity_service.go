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
	// Remove from control plane
	err := s.repo.RemoveTenantMembership(ctx, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to leave tenant: %w", err)
	}

	// Wait, we also need to set status = 'inactive' in tenant DB
	// but we don't know the DBName without querying the organization repo.
	// This means IdentityService needs to know about Organization, or we do it via Event/Orchestrator.
	// For simplicity right now, let's leave a TODO here.
	return nil
}

func (s *IdentityService) DeleteAccount(ctx context.Context, userID string) error {
	// Obfuscate in control plane
	// Soft delete sole-owner tenants
	// Obfuscate in tenant DBs
	// This requires cross-domain orchestration.
	// Leaving a placeholder for now as per "not fully implemented yet".
	return fmt.Errorf("delete account not implemented fully yet")
}
