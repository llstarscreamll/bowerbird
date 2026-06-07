package application

import (
	"github.com/bowerbird/internal/identity/application/commands"
	"github.com/bowerbird/internal/identity/application/queries"
	"github.com/bowerbird/internal/identity/domain"
	"github.com/bowerbird/internal/platform/auth"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	Auth          *commands.AuthService
	LeaveTenant   *commands.LeaveTenantCommand
	DeleteAccount *commands.DeleteAccountCommand
}

type Queries struct {
	ListUserTenants *queries.ListUserTenantsQuery
}

func NewApplication(repo domain.Repository, tokenGen *auth.TokenGenerator, appEnv string) *Application {
	return &Application{
		Commands: Commands{
			Auth:          commands.NewAuthService(repo, tokenGen, appEnv),
			LeaveTenant:   commands.NewLeaveTenantCommand(repo),
			DeleteAccount: commands.NewDeleteAccountCommand(repo),
		},
		Queries: Queries{
			ListUserTenants: queries.NewListUserTenantsQuery(repo),
		},
	}
}
