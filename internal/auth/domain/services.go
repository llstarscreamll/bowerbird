package domain

import "context"

type AuthServer interface {
	GetLoginUrl() string
	GetUserInfo(ctx context.Context, authCode string) (User, error)
}
