package domain

import "context"

type UserRepository interface {
	Upsert(ctx context.Context, user User) error
}

type SessionRepository interface {
	Save(userID, sessionID string) error
}
