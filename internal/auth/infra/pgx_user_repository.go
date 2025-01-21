package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxUserRepository) Upsert(ctx context.Context, user domain.User) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO users (id, first_name, last_name, email, photo_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO NOTHING`,
		user.ID, user.GivenName, user.FamilyName, user.Email, user.PictureUrl,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *PgxUserRepository) GetByID(ctx context.Context, ID string) (domain.User, error) {
	var u domain.User

	row := r.pool.QueryRow(
		ctx,
		`SELECT id, email, first_name, last_name, photo_url FROM users WHERE id = $1`,
		ID,
	)

	row.Scan(&u.ID, &u.Email, &u.GivenName, &u.FamilyName, &u.PictureUrl)

	if u.ID != "" {
		u.Name = u.GivenName + " " + u.FamilyName
	}

	return u, nil
}

func NewPgxUserRepository(pool *pgxpool.Pool) *PgxUserRepository {
	return &PgxUserRepository{pool}
}
