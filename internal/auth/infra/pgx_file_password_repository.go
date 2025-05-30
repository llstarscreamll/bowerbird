package infra

import (
	"context"
	"errors"
	"strings"

	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxFilePasswordRepository struct {
	pool  *pgxpool.Pool
	crypt commonDomain.Crypt
}

func (r PgxFilePasswordRepository) GetByUserID(ctx context.Context, userID string) ([]string, error) {
	passwords := make([]string, 0)

	row := r.pool.QueryRow(
		ctx,
		`SELECT passwords FROM file_passwords WHERE user_id = $1`,
		userID,
	)

	var encryptedPasswords string
	if err := row.Scan(&encryptedPasswords); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return passwords, err
	}

	decryptedPasswords, err := r.crypt.DecryptString(encryptedPasswords)
	if err != nil {
		return passwords, err
	}

	return strings.Split(decryptedPasswords, "\n"), nil
}

func (r PgxFilePasswordRepository) Upsert(ctx context.Context, userID string, passwords []string) error {
	encryptedPasswords, err := r.crypt.EncryptString(strings.Join(passwords, "\n"))
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(
		ctx,
		`INSERT INTO file_passwords (user_id, passwords)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET passwords = $2`,
		userID,
		encryptedPasswords,
	)

	return err
}

func NewPgxFilePasswordRepository(pool *pgxpool.Pool, crypt commonDomain.Crypt) *PgxFilePasswordRepository {
	return &PgxFilePasswordRepository{pool, crypt}
}
