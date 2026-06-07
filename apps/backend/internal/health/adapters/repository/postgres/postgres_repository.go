package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) Ping(ctx context.Context) error {
	const query = `SELECT 1`

	var value int
	return r.db.QueryRow(ctx, query).Scan(&value)
}
