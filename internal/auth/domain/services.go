package domain

import (
	"context"
)

type AuthServerGateway interface {
	GetLoginUrl(provider string, scopes []string) (string, error)
	GetTokens(ctx context.Context, provider string, authCode string) (Tokens, error)
	GetUserProfile(ctx context.Context, provider string, authCode string) (User, error)
}

type AuthServerStrategy interface {
	GetLoginUrl(scopes []string) (string, error)
	GetTokens(ctx context.Context, authCode string) (Tokens, error)
	GetUserProfile(ctx context.Context, authCode string) (User, error)
}
