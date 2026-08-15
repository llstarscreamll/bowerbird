package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/domain"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/id"
)

var (
	ErrLocalAuthDisabled  = errors.New("local auth is disabled in this environment")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email is not verified")
	ErrInvalidEmail       = errors.New("invalid email")
)

type AuthService struct {
	repo           ports.Repository
	tokenGen       *auth.TokenGenerator
	refreshStore   auth.RefreshTokenStore
	localEnabled   bool
	operatorEmails []string
}

func NewAuthService(
	repo ports.Repository,
	tokenGen *auth.TokenGenerator,
	refreshStore auth.RefreshTokenStore,
	appEnv string,
	operatorEmails []string,
) *AuthService {
	return &AuthService{
		repo:           repo,
		tokenGen:       tokenGen,
		refreshStore:   refreshStore,
		localEnabled:   appEnv == "local" || appEnv == "development",
		operatorEmails: operatorEmails,
	}
}

func (s *AuthService) issueTokens(ctx context.Context, user *domain.User) (*auth.TokenPair, error) {
	if err := EnsureOperatorRole(ctx, s.repo, s.operatorEmails, user); err != nil {
		return nil, err
	}
	tokens, err := s.tokenGen.GenerateTokens(user.ID, user.Email, user.FirstName, user.LastName, user.PictureURL)
	if err != nil {
		return nil, err
	}
	if s.refreshStore != nil {
		expiresAt := time.Now().Add(s.tokenGen.RefreshTTL())
		if err := s.refreshStore.Save(ctx, tokens.RefreshJTI, user.ID, expiresAt); err != nil {
			return nil, fmt.Errorf("persist refresh token: %w", err)
		}
	}
	return tokens, nil
}

func (s *AuthService) RegisterLocal(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	if !s.localEnabled {
		return nil, ErrLocalAuthDisabled
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}
	hashed, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(id.NewULID(), email, "Local", "User", "")
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	identity := domain.NewUserIdentity(id.NewULID(), user.ID, "local", hashed)
	err = s.repo.CreateUserIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) LoginLocal(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	if !s.localEnabled {
		return nil, ErrLocalAuthDisabled
	}

	user, err := s.repo.FindUserByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	identity, err := s.repo.FindUserIdentityByProvider(ctx, user.ID, "local")
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := auth.CheckPassword(identity.ProviderID, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) OAuthLogin(ctx context.Context, email, provider, providerID, name, pictureURL string, emailVerified bool) (*auth.TokenPair, error) {
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

	existingUser, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if existingUser != nil {
		user = existingUser
		_, err = s.repo.FindUserIdentityByProvider(ctx, user.ID, provider)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				identity := domain.NewUserIdentity(id.NewULID(), user.ID, provider, providerID)
				err = s.repo.CreateUserIdentity(ctx, identity)
				if err != nil {
					return nil, fmt.Errorf("failed to link identity: %w", err)
				}
			} else {
				return nil, err
			}
		}
	} else {
		user = domain.NewUser(id.NewULID(), email, firstName, lastName, pictureURL)
		err = s.repo.CreateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		identity := domain.NewUserIdentity(id.NewULID(), user.ID, provider, providerID)
		err = s.repo.CreateUserIdentity(ctx, identity)
		if err != nil {
			return nil, fmt.Errorf("failed to create identity: %w", err)
		}
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	userID, jti, err := s.tokenGen.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	if s.refreshStore != nil {
		storedUserID, err := s.refreshStore.Consume(ctx, jti)
		if err != nil {
			return nil, err
		}
		if storedUserID != userID {
			return nil, auth.ErrInvalidToken
		}
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if s.refreshStore == nil {
		return nil
	}
	_, jti, err := s.tokenGen.ValidateRefreshToken(refreshToken)
	if err != nil {
		// Cookie may already be expired; treat as logged out.
		return nil
	}
	return s.refreshStore.Revoke(ctx, jti)
}

func (s *AuthService) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	if s.refreshStore == nil {
		return nil
	}
	return s.refreshStore.RevokeAllForUser(ctx, userID)
}
