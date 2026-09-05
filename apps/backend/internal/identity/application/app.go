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
	RegisterLocal          *commands.RegisterLocalCommand
	LoginLocal             *commands.LoginLocalCommand
	OAuthLogin             *commands.OAuthLoginCommand
	RefreshToken           *commands.RefreshTokenCommand
	RevokeRefreshToken     *commands.RevokeRefreshTokenCommand
	RevokeAllRefreshTokens *commands.RevokeAllRefreshTokensCommand
	LeaveTenant            *commands.LeaveTenantCommand
	DeleteAccount          *commands.DeleteAccountCommand
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
	if repo == nil {
		panic("identity repository is required")
	}
	if tokenGen == nil {
		panic("token generator is required")
	}
	if refreshStore == nil {
		panic("refresh token store is required")
	}
	return &Application{
		Commands: Commands{
			RegisterLocal:          commands.NewRegisterLocalCommand(repo, tokenGen, refreshStore, appEnv, operatorEmails),
			LoginLocal:             commands.NewLoginLocalCommand(repo, tokenGen, refreshStore, appEnv, operatorEmails),
			OAuthLogin:             commands.NewOAuthLoginCommand(repo, tokenGen, refreshStore, operatorEmails),
			RefreshToken:           commands.NewRefreshTokenCommand(repo, tokenGen, refreshStore, operatorEmails),
			RevokeRefreshToken:     commands.NewRevokeRefreshTokenCommand(tokenGen, refreshStore),
			RevokeAllRefreshTokens: commands.NewRevokeAllRefreshTokensCommand(refreshStore),
			LeaveTenant:            commands.NewLeaveTenantCommand(repo),
			DeleteAccount:          commands.NewDeleteAccountCommand(repo),
		},
		Queries: Queries{
			ListUserTenants: queries.NewListUserTenantsQuery(repo),
		},
		Identity: NewIdentityService(repo, operatorEmails),
	}
}

func (a *Application) IsPlatformOperator(ctx context.Context, userID string) (bool, error) {
	return a.Identity.IsPlatformOperator(ctx, userID)
}
