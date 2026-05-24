package application

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/money-path/bowerbird/apps/backend/internal/identity/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
)

type AuthService struct {
	repo         domain.Repository
	tokenGen     *auth.TokenGenerator
	localEnabled bool
}

func NewAuthService(repo domain.Repository, tokenGen *auth.TokenGenerator, appEnv string) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenGen:     tokenGen,
		localEnabled: appEnv == "local" || appEnv == "development",
	}
}

// RegisterLocal is only for local dev/e2e testing
func (s *AuthService) RegisterLocal(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	if !s.localEnabled {
		return nil, errors.New("local auth is disabled in this environment")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := domain.NewUser("", email, "Local", "User", "")
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Fetch to get DB-generated ID if needed, or assume repository assigns it.
	// Since we use gen_random_uuid(), we should fetch it back or generate uuid in code.
	// Wait, our repository logic for CreateUser doesn't fetch the generated ID.
	// We need to modify CreateUser to scan the ID, or we generate UUID in Go.
	// Let's assume we update repo to return the generated User or generate UUID in Go.
	// Let's fetch it by email.
	createdUser, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Create identity
	identity := domain.NewUserIdentity("", createdUser.ID, "local", string(hashed))
	err = s.repo.CreateUserIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}

	return s.tokenGen.GenerateTokens(createdUser.ID, createdUser.Email, createdUser.FirstName, createdUser.LastName, createdUser.PictureURL)
}

// LoginLocal is only for local dev/e2e testing
func (s *AuthService) LoginLocal(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	if !s.localEnabled {
		return nil, errors.New("local auth is disabled in this environment")
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	identity, err := s.repo.FindUserIdentityByProvider(ctx, user.ID, "local")
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(identity.ProviderID), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return s.tokenGen.GenerateTokens(user.ID, user.Email, user.FirstName, user.LastName, user.PictureURL)
}

// OAuthLogin handles login or registration via OAuth provider
func (s *AuthService) OAuthLogin(ctx context.Context, email, provider, providerID, name, pictureURL string) (*auth.TokenPair, error) {
	var user *domain.User

	// Simple heuristic to split name (since providers often just give "displayName" or "name")
	firstName := name
	lastName := ""
	// For production, you'd want a better heuristic or fetch family_name if available.

	// Check if user exists by email (Account Linking)
	existingUser, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if existingUser != nil {
		user = existingUser
		// Ensure identity exists
		_, err = s.repo.FindUserIdentityByProvider(ctx, user.ID, provider)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				// Link new identity
				identity := domain.NewUserIdentity("", user.ID, provider, providerID)
				err = s.repo.CreateUserIdentity(ctx, identity)
				if err != nil {
					return nil, fmt.Errorf("failed to link identity: %w", err)
				}
			} else {
				return nil, err
			}
		}
	} else {
		// Create new user
		user = domain.NewUser("", email, firstName, lastName, pictureURL)
		err = s.repo.CreateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		user, err = s.repo.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}

		identity := domain.NewUserIdentity("", user.ID, provider, providerID)
		err = s.repo.CreateUserIdentity(ctx, identity)
		if err != nil {
			return nil, fmt.Errorf("failed to create identity: %w", err)
		}
	}

	return s.tokenGen.GenerateTokens(user.ID, user.Email, user.FirstName, user.LastName, user.PictureURL)
}

// RefreshToken validates a refresh token and issues a new pair
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	userID, err := s.tokenGen.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.tokenGen.GenerateTokens(user.ID, user.Email, user.FirstName, user.LastName, user.PictureURL)
}
