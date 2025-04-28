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
		`SELECT id, name, color, icon
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

		err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Icon)
		if err != nil {
			return nil, err
		}

		categories = append(categories, c)
	}

	return categories, nil
}

func (r *PgxCategoryRepository) Create(ctx context.Context, category domain.Category) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO categories (id, wallet_id, name, color, icon, created_by_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		category.ID, category.WalletID, category.Name, category.Color, category.Icon, category.CreatedByID, category.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

func NewPgxCategoryRepository(pool *pgxpool.Pool) *PgxCategoryRepository {
	return &PgxCategoryRepository{pool: pool}
}
