package commands

import (
	"context"

	"github.com/bowerbird/internal/platform/auth"
)

type RevokeRefreshTokenCommand struct {
	tokenGen     *auth.TokenGenerator
	refreshStore auth.RefreshTokenStore
}

func NewRevokeRefreshTokenCommand(tokenGen *auth.TokenGenerator, refreshStore auth.RefreshTokenStore) *RevokeRefreshTokenCommand {
	return &RevokeRefreshTokenCommand{
		tokenGen:     tokenGen,
		refreshStore: refreshStore,
	}
}

func (cmd *RevokeRefreshTokenCommand) Execute(ctx context.Context, refreshToken string) error {
	_, jti, err := cmd.tokenGen.ValidateRefreshToken(refreshToken)
	if err != nil {
		// Cookie may already be expired; treat as logged out.
		return nil
	}
	return cmd.refreshStore.Revoke(ctx, jti)
}
