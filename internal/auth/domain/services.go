package domain

import (
	"context"
	"time"
)

type AuthServerGateway interface {
	GetLoginUrl(provider string, scopes []string) (string, error)
	GetTokens(ctx context.Context, provider string, authCode string) (string, string, time.Time, error)
	GetUserProfile(ctx context.Context, provider string, authCode string) (User, error)
}
