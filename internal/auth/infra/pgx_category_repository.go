package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxCategoryRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxCategoryRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.Category, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, name, color
		FROM categories 
		WHERE wallet_id = $1`,
		walletID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	categories := make([]domain.Category, 0)

	for rows.Next() {
		c := domain.Category{}

		err := rows.Scan(&c.ID, &c.Name, &c.Color)
		if err != nil {
			return nil, err
		}

		categories = append(categories, c)
	}

	return categories, nil
}

func NewPgxCategoryRepository(pool *pgxpool.Pool) *PgxCategoryRepository {
	return &PgxCategoryRepository{pool: pool}
}
