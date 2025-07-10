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
		ON CONFLICT (wallet_id, mail_address) DO UPDATE SET access_token = $6, refresh_token = $7`,
		ID, userID, walletID, mailProvider, mailAddress, accessToken, refreshToken, expiresAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *PgxMailCredentialRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.MailCredential, error) {
	var credentials []domain.MailCredential

	rows, err := r.pool.Query(
		ctx,
		`SELECT id, wallet_id, user_id, mail_provider, mail_address, access_token, refresh_token, expires_at, last_read_at
		FROM mail_credentials
		WHERE wallet_id = $1`,
		walletID,
	)
	if err != nil {
		return credentials, err
	}

	defer rows.Close()
	for rows.Next() {
		credential := domain.MailCredential{}

		err := rows.Scan(&credential.ID, &credential.WalletID, &credential.UserID, &credential.MailProvider, &credential.MailAddress, &credential.AccessToken, &credential.RefreshToken, &credential.ExpiresAt, &credential.LastReadAt)
		if err != nil {
			return nil, err
		}

		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func (r *PgxMailCredentialRepository) UpdateLastReadAt(ctx context.Context, ID string, lastReadAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE mail_credentials SET last_read_at = $1 WHERE id = $2`, lastReadAt, ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *PgxMailCredentialRepository) Delete(ctx context.Context, ID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mail_credentials WHERE id = $1`, ID)
	if err != nil {
		return err
	}

	return nil
}

func NewPgxMailCredentialRepository(pool *pgxpool.Pool) *PgxMailCredentialRepository {
	return &PgxMailCredentialRepository{pool}
}
