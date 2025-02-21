package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxWalletRepository struct {
	pool *pgxpool.Pool
}

func (r PgxWalletRepository) Create(ctx context.Context, wallet domain.UserWallet) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO wallets (id, name, created_at)
		VALUES ($1, $2, $3)`,
		wallet.ID, wallet.Name, wallet.CreatedAt,
	)

	if err != nil {
		return err
	}

	_, err = r.pool.Exec(
		ctx,
		`INSERT INTO user_has_wallets (user_id, wallet_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`,
		wallet.UserID, wallet.ID, wallet.Role, wallet.JoinedAt,
	)

	return err

}

func (r *PgxWalletRepository) FindByUserID(ctx context.Context, userID string) ([]domain.UserWallet, error) {
	var wallets []domain.UserWallet

	rows, err := r.pool.Query(
		ctx,
		`SELECT w.id, w.name, uw."role", uw.joined_at, w.created_at
		FROM wallets w
		JOIN user_has_wallets uw ON w.id = uw.wallet_id
		WHERE uw.user_id = $1`,
		userID,
	)

	if err != nil {
		return wallets, err
	}

	defer rows.Close()

	for rows.Next() {
		w := domain.UserWallet{}

		err := rows.Scan(&w.ID, &w.Name, &w.Role, &w.JoinedAt, &w.CreatedAt)
		if err != nil {
			return nil, err
		}

		wallets = append(wallets, w)
	}

	return wallets, nil
}

func NewPgxWalletRepository(pool *pgxpool.Pool) *PgxWalletRepository {
	return &PgxWalletRepository{pool}
}
