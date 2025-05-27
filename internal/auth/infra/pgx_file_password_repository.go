package infra

import (
	"context"
	"strings"

	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

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
	if err := row.Scan(&encryptedPasswords); err != nil {
		return passwords, err
	}

	decryptedPasswords, err := r.crypt.DecryptString(encryptedPasswords)
	if err != nil {
		return passwords, err
	}

	return strings.Split(decryptedPasswords, "\n"), nil
}

func NewPgxFilePasswordRepository(pool *pgxpool.Pool, crypt commonDomain.Crypt) *PgxFilePasswordRepository {
	return &PgxFilePasswordRepository{pool, crypt}
}
