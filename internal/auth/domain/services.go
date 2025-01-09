package domain

import "context"

type AuthServer interface {
	GetLoginUrl(scopes []string) string
	GetUserInfo(ctx context.Context, authCode string) (User, error)
}

type Crypt interface {
	EncryptString(str string) string
}
