package application

import (
	"context"

	"github.com/bowerbird/internal/identity/application/commands"
	"github.com/bowerbird/internal/identity/application/queries"
	"github.com/bowerbird/internal/identity/domain"
	"github.com/bowerbird/internal/platform/auth"
)

type Application struct {
	Commands Commands
	Queries  Queries
	Identity *IdentityService
}

type Commands struct {
	Auth          *commands.AuthService
	LeaveTenant   *commands.LeaveTenantCommand
	DeleteAccount *commands.DeleteAccountCommand
}

type Queries struct {
	ListUserTenants *queries.ListUserTenantsQuery
}

func NewApplication(
	repo domain.Repository,
	tokenGen *auth.TokenGenerator,
	refreshStore auth.RefreshTokenStore,
	appEnv string,
	operatorEmails []string,
) *Application {
	return &Application{
		Commands: Commands{
			Auth:          commands.NewAuthService(repo, tokenGen, refreshStore, appEnv, operatorEmails),
			LeaveTenant:   commands.NewLeaveTenantCommand(repo),
			DeleteAccount: commands.NewDeleteAccountCommand(repo),
		},
		Queries: Queries{
			ListUserTenants: queries.NewListUserTenantsQuery(repo),
		},
		Identity: NewIdentityService(repo, operatorEmails),
	}
}

func (a *Application) IsPlatformOperator(ctx context.Context, userID string) (bool, error) {
	if a.Identity == nil {
		return false, nil
	}
	return a.Identity.IsPlatformOperator(ctx, userID)
}
