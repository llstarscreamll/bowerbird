package infra

import "github.com/jackc/pgx/v5/pgxpool"

type PgxSessionRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxSessionRepository) Save(userID, sessionID string) error {
	return nil
}

func NewPgxSessionRepository(pool *pgxpool.Pool) *PgxSessionRepository {
	return &PgxSessionRepository{pool}
}
