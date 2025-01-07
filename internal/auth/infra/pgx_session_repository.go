package infra

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxSessionRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxSessionRepository) Save(ctx context.Context, sessionID, userID string, expirationDate time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`, sessionID, userID, expirationDate)
	if err != nil {
		return err
	}

	return nil
}

func NewPgxSessionRepository(pool *pgxpool.Pool) *PgxSessionRepository {
	return &PgxSessionRepository{pool}
}
