package commands

import (
	"context"

	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/platform/auth"
)

type RefreshTokenCommand struct {
	repo         ports.Repository
	tokenGen     *auth.TokenGenerator
	refreshStore auth.RefreshTokenStore
	sessions     *sessionIssuer
}

func NewRefreshTokenCommand(
	repo ports.Repository,
	tokenGen *auth.TokenGenerator,
	refreshStore auth.RefreshTokenStore,
	operatorEmails []string,
) *RefreshTokenCommand {
	return &RefreshTokenCommand{
		repo:         repo,
		tokenGen:     tokenGen,
		refreshStore: refreshStore,
		sessions:     newSessionIssuer(repo, tokenGen, refreshStore, operatorEmails),
	}
}

func (cmd *RefreshTokenCommand) Execute(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	userID, jti, err := cmd.tokenGen.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	storedUserID, err := cmd.refreshStore.Consume(ctx, jti)
	if err != nil {
		return nil, err
	}
	if storedUserID != userID {
		return nil, auth.ErrInvalidToken
	}

	user, err := cmd.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return cmd.sessions.issue(ctx, user)
}
