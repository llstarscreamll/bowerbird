package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxMailCredentialRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxMailCredentialRepository) Save(ctx context.Context, ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken string, expiresAt time.Time) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO mail_credentials (id, user_id, wallet_id, mail_provider, mail_address, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, mail_address) DO UPDATE SET access_token = $5, refresh_token = $6`,
		ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken, expiresAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *PgxMailCredentialRepository) FindByUserID(ctx context.Context, userID string) ([]domain.MailCredential, error) {
	var credentials []domain.MailCredential

	rows, err := r.pool.Query(
		ctx,
		`SELECT id, user_id, mail_provider, mail_address, access_token, refresh_token, expires_at
		FROM mail_credentials
		WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return credentials, err
	}

	defer rows.Close()
	for rows.Next() {
		credential := domain.MailCredential{}

		err := rows.Scan(&credential.ID, &credential.UserID, &credential.MailProvider, &credential.MailAddress, &credential.AccessToken, &credential.RefreshToken, &credential.ExpiresAt)
		if err != nil {
			return nil, err
		}

		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func NewPgxMailCredentialRepository(pool *pgxpool.Pool) *PgxMailCredentialRepository {
	return &PgxMailCredentialRepository{pool}
}
