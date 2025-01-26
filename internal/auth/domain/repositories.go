package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Upsert(ctx context.Context, user User) error
	GetByID(ctx context.Context, ID string) (User, error)
}

type SessionRepository interface {
	Save(ctx context.Context, userID, sessionID string, expirationDate time.Time) error
	GetByID(ctx context.Context, ID string) (string, error)
}

type MailCredentialRepository interface {
	Save(ctx context.Context, ID, userID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error
}
