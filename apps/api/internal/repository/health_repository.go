package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepository struct {
	db *pgxpool.Pool
}

func NewHealthRepository(db *pgxpool.Pool) HealthRepository {
	return HealthRepository{db: db}
}

func (r HealthRepository) IsDatabaseHealthy(ctx context.Context) error {
	const query = `SELECT 1`

	var value int
	return r.db.QueryRow(ctx, query).Scan(&value)
}
