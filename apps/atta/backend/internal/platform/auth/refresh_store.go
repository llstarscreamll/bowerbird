package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRefreshNotFound = errors.New("refresh token not found")
	ErrRefreshRevoked  = errors.New("refresh token revoked")
	ErrRefreshExpired  = errors.New("refresh token expired")
)

// RefreshTokenStore persists refresh-token JTIs for rotation and revocation.
type RefreshTokenStore interface {
	Save(ctx context.Context, jti, userID string, expiresAt time.Time) error
	// Consume validates an active token and revokes it in one step (rotation).
	Consume(ctx context.Context, jti string) (userID string, err error)
	Revoke(ctx context.Context, jti string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}
