package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Upsert(ctx context.Context, user User) error
}

type SessionRepository interface {
	Save(ctx context.Context, userID, sessionID string, expirationDate time.Time) error
}
