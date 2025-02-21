package infra

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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

func (r *PgxSessionRepository) GetByID(ctx context.Context, ID string) (string, error) {
	var userID string

	row := r.pool.QueryRow(
		ctx,
		`SELECT user_id FROM sessions WHERE id = $1`,
		ID,
	)

	err := row.Scan(&userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	return userID, nil
}

func (r *PgxSessionRepository) Delete(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}

	return nil
}

func NewPgxSessionRepository(pool *pgxpool.Pool) *PgxSessionRepository {
	return &PgxSessionRepository{pool}
}
