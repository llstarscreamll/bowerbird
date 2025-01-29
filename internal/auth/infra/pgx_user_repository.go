package infra

import (
	"context"
	"errors"
	"llstarscreamll/bowerbird/internal/auth/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxUserRepository) Upsert(ctx context.Context, user domain.User) (string, error) {
	row := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (id, first_name, last_name, email, photo_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET last_login_at = NOW()
		RETURNING id`,
		user.ID, user.GivenName, user.FamilyName, user.Email, user.PictureUrl,
	)

	var id string

	err := row.Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	return id, nil
}

func (r *PgxUserRepository) GetByID(ctx context.Context, ID string) (domain.User, error) {
	var u domain.User

	row := r.pool.QueryRow(
		ctx,
		`SELECT id, email, first_name, last_name, photo_url FROM users WHERE id = $1`,
		ID,
	)

	err := row.Scan(&u.ID, &u.Email, &u.GivenName, &u.FamilyName, &u.PictureUrl)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, err
	}

	if u.ID != "" {
		u.Name = u.GivenName + " " + u.FamilyName
	}

	return u, nil
}

func NewPgxUserRepository(pool *pgxpool.Pool) *PgxUserRepository {
	return &PgxUserRepository{pool}
}
