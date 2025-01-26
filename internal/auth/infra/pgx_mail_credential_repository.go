package infra

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxMailCredentialRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxMailCredentialRepository) Save(ctx context.Context, ID, userID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO mail_credentials (id, user_id, mail_provider, mail_address, access_token, refresh_token, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ID, userID, mailProvider, mailAddress, accessToken, refreshToken, expiresAt)
	if err != nil {
		return err
	}

	return nil
}

func NewPgxMailCredentialRepository(pool *pgxpool.Pool) *PgxMailCredentialRepository {
	return &PgxMailCredentialRepository{pool}
}
