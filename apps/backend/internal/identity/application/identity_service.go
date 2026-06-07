package application

import (
	"context"

	"github.com/bowerbird/internal/identity/application/commands"
	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/application/queries"
)

type IdentityService struct {
	listUserTenants *queries.ListUserTenantsQuery
	leaveTenant     *commands.LeaveTenantCommand
	deleteAccount   *commands.DeleteAccountCommand
}

func NewIdentityService(repo ports.Repository) *IdentityService {
	return &IdentityService{
		listUserTenants: queries.NewListUserTenantsQuery(repo),
		leaveTenant:     commands.NewLeaveTenantCommand(repo),
		deleteAccount:   commands.NewDeleteAccountCommand(repo),
	}
}

type TenantMembershipDTO = queries.TenantMembershipDTO

func (s *IdentityService) ListUserTenants(ctx context.Context, userID string) ([]TenantMembershipDTO, error) {
	return s.listUserTenants.Execute(ctx, userID)
}

func (s *IdentityService) LeaveTenant(ctx context.Context, userID, tenantID string) error {
	return s.leaveTenant.Execute(ctx, userID, tenantID)
}

func (s *IdentityService) DeleteAccount(ctx context.Context, userID string) error {
	return s.deleteAccount.Execute(ctx, userID)
}
