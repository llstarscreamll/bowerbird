package application

import (
	"context"

	"github.com/bowerbird/internal/identity/application/commands"
	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/application/queries"
)

type IdentityService struct {
	repo            ports.Repository
	operatorEmails  []string
	listUserTenants *queries.ListUserTenantsQuery
	leaveTenant     *commands.LeaveTenantCommand
	deleteAccount   *commands.DeleteAccountCommand
}

func NewIdentityService(repo ports.Repository, operatorEmails []string) *IdentityService {
	return &IdentityService{
		repo:            repo,
		operatorEmails:  operatorEmails,
		listUserTenants: queries.NewListUserTenantsQuery(repo),
		leaveTenant:     commands.NewLeaveTenantCommand(repo),
		deleteAccount:   commands.NewDeleteAccountCommand(repo),
	}
}

type TenantMembershipDTO = queries.TenantMembershipDTO

type MeDTO struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	PictureURL       string `json:"picture_url"`
	PlatformOperator bool   `json:"platform_operator"`
}

func (s *IdentityService) ListUserTenants(ctx context.Context, userID string) ([]TenantMembershipDTO, error) {
	return s.listUserTenants.Execute(ctx, userID)
}

func (s *IdentityService) LeaveTenant(ctx context.Context, userID, tenantID string) error {
	return s.leaveTenant.Execute(ctx, userID, tenantID)
}

func (s *IdentityService) DeleteAccount(ctx context.Context, userID string) error {
	return s.deleteAccount.Execute(ctx, userID)
}

func (s *IdentityService) GetMe(ctx context.Context, userID string) (*MeDTO, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := commands.EnsureOperatorRole(ctx, s.repo, s.operatorEmails, user); err != nil {
		return nil, err
	}
	return &MeDTO{
		ID:               user.ID,
		Email:            user.Email,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		PictureURL:       user.PictureURL,
		PlatformOperator: user.IsPlatformOperator(),
	}, nil
}

func (s *IdentityService) IsPlatformOperator(ctx context.Context, userID string) (bool, error) {
	me, err := s.GetMe(ctx, userID)
	if err != nil {
		return false, err
	}
	return me.PlatformOperator, nil
}
