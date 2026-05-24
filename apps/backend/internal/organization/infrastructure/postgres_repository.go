package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/money-path/bowerbird/apps/backend/internal/organization/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `
		INSERT INTO tenants (organization_name, slug, db_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		org.Name,
		org.Slug,
		org.DBName,
		org.Status,
		org.CreatedAt,
		org.UpdatedAt,
	).Scan(&org.ID)

	return err
}

func (r *PostgresRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) AddMembership(ctx context.Context, userID, tenantID, role string) error {
	query := `INSERT INTO tenant_memberships (user_id, tenant_id, role) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, userID, tenantID, role)
	return err
}
