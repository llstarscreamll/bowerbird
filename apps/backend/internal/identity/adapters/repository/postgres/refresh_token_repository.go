package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bowerbird/internal/platform/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (jti, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, jti, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) Consume(ctx context.Context, jti string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE jti = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING user_id
	`, jti).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", auth.ErrRefreshNotFound
	}
	if err != nil {
		return "", fmt.Errorf("consume refresh token: %w", err)
	}
	return userID, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, jti string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW() WHERE jti = $1 AND revoked_at IS NULL
	`, jti)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}
