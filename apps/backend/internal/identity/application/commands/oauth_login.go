package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/domain"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/id"
)

type OAuthLoginCommand struct {
	repo     ports.Repository
	sessions *sessionIssuer
}

func NewOAuthLoginCommand(
	repo ports.Repository,
	tokenGen *auth.TokenGenerator,
	refreshStore auth.RefreshTokenStore,
	operatorEmails []string,
) *OAuthLoginCommand {
	return &OAuthLoginCommand{
		repo:     repo,
		sessions: newSessionIssuer(repo, tokenGen, refreshStore, operatorEmails),
	}
}

func (cmd *OAuthLoginCommand) Execute(ctx context.Context, email, provider, providerID, name, pictureURL string, emailVerified bool) (*auth.TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrEmailNotVerified
	}
	if !emailVerified {
		return nil, ErrEmailNotVerified
	}

	var user *domain.User
	firstName := name
	lastName := ""

	existingUser, err := cmd.repo.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if existingUser != nil {
		user = existingUser
		_, err = cmd.repo.FindUserIdentityByProvider(ctx, user.ID, provider)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				identity := domain.NewUserIdentity(id.NewULID(), user.ID, provider, providerID)
				if err = cmd.repo.CreateUserIdentity(ctx, identity); err != nil {
					return nil, fmt.Errorf("failed to link identity: %w", err)
				}
			} else {
				return nil, err
			}
		}
	} else {
		user = domain.NewUser(id.NewULID(), email, firstName, lastName, pictureURL)
		if err = cmd.repo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		identity := domain.NewUserIdentity(id.NewULID(), user.ID, provider, providerID)
		if err = cmd.repo.CreateUserIdentity(ctx, identity); err != nil {
			return nil, fmt.Errorf("failed to create identity: %w", err)
		}
	}

	return cmd.sessions.issue(ctx, user)
}
