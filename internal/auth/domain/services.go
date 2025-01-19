package domain

import (
	"context"
	"time"
)

type AuthServer interface {
	GetLoginUrl(scopes []string) string
	GetTokens(ctx context.Context, authCode string) (string, string, time.Time, error)
	GetUserInfo(ctx context.Context, authCode string) (User, error)
}
