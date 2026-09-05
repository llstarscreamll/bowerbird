package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/domain"
	"github.com/bowerbird/internal/platform/auth"
)

type sessionIssuer struct {
	repo           ports.Repository
	tokenGen       *auth.TokenGenerator
	refreshStore   auth.RefreshTokenStore
	operatorEmails []string
}

func newSessionIssuer(
	repo ports.Repository,
	tokenGen *auth.TokenGenerator,
	refreshStore auth.RefreshTokenStore,
	operatorEmails []string,
) *sessionIssuer {
	return &sessionIssuer{
		repo:           repo,
		tokenGen:       tokenGen,
		refreshStore:   refreshStore,
		operatorEmails: operatorEmails,
	}
}

func (s *sessionIssuer) issue(ctx context.Context, user *domain.User) (*auth.TokenPair, error) {
	if err := EnsureOperatorRole(ctx, s.repo, s.operatorEmails, user); err != nil {
		return nil, err
	}
	tokens, err := s.tokenGen.GenerateTokens(user.ID, user.Email, user.FirstName, user.LastName, user.PictureURL)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.tokenGen.RefreshTTL())
	if err := s.refreshStore.Save(ctx, tokens.RefreshJTI, user.ID, expiresAt); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}
	return tokens, nil
}
