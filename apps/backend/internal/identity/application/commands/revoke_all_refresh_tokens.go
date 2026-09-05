package commands

import (
	"context"

	"github.com/bowerbird/internal/platform/auth"
)

type RevokeAllRefreshTokensCommand struct {
	refreshStore auth.RefreshTokenStore
}

func NewRevokeAllRefreshTokensCommand(refreshStore auth.RefreshTokenStore) *RevokeAllRefreshTokensCommand {
	return &RevokeAllRefreshTokensCommand{refreshStore: refreshStore}
}

func (cmd *RevokeAllRefreshTokensCommand) Execute(ctx context.Context, userID string) error {
	return cmd.refreshStore.RevokeAllForUser(ctx, userID)
}
