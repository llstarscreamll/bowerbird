package domain

import (
	"context"
	"time"
)

type AuthServerGateway interface {
	GetLoginUrl(scopes []string) (string, error)
	GetTokens(ctx context.Context, authCode string) (string, string, time.Time, error)
	GetUserProfile(ctx context.Context, authCode string) (User, error)
}
