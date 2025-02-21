package infra

import (
	"context"
	"errors"

	"llstarscreamll/bowerbird/internal/auth/domain"
)

type AuthServerGateway struct {
	authStrategies map[string]domain.AuthServerStrategy
}

// ToDo: state should be stored somewhere and be validated on callback to prevent CSRF attacks
func (g AuthServerGateway) GetLoginUrl(provider, redirectUrl string, scopes []string, state string) (string, error) {
	strategy, ok := g.authStrategies[provider]
	if !ok {
		return "", errors.New("OAuth provider not supported")
	}

	return strategy.GetLoginUrl(redirectUrl, scopes, state)
}

func (g AuthServerGateway) GetTokens(ctx context.Context, provider string, authCode, state string) (domain.Tokens, error) {
	strategy, ok := g.authStrategies[provider]
	if !ok {
		return domain.Tokens{}, errors.New("OAuth provider not supported")
	}

	return strategy.GetTokens(ctx, authCode, state)
}

// ToDo: state should be validated to prevent CSRF attacks
func (g AuthServerGateway) GetUserProfile(ctx context.Context, provider string, accessToken string) (domain.User, error) {
	strategy, ok := g.authStrategies[provider]
	if !ok {
		return domain.User{}, errors.New("OAuth provider not supported")
	}

	return strategy.GetUserProfile(ctx, accessToken)
}

func NewAuthServerGateway(googleAuth GoogleAuthStrategy, microsoftStrategy MicrosoftAuthStrategy) *AuthServerGateway {
	return &AuthServerGateway{
		authStrategies: map[string]domain.AuthServerStrategy{
			"google":    googleAuth,
			"microsoft": microsoftStrategy,
		},
	}
}
