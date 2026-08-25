package postgres

import (
	"context"
	"time"

	"github.com/bowerbird/internal/tenant/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, org *domain.Tenant) error {
	query := `
		INSERT INTO tenants (id, tenant_name, slug, db_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.DBName,
		org.Status,
		org.CreatedAt,
		org.UpdatedAt,
	)

	return err
}

func (r *PostgresRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, userID string) (*domain.Tenant, error) {
	query := `
		SELECT t.id, t.tenant_name, t.slug, t.db_name, t.status, t.created_at, t.updated_at,
		       (SELECT COUNT(*) FROM tenant_memberships WHERE tenant_id = t.id AND deleted_at IS NULL) as members_count,
			   tm.role
		FROM tenants t
		INNER JOIN tenant_memberships tm
			ON tm.tenant_id = t.id AND tm.user_id = $2 AND tm.deleted_at IS NULL
		WHERE t.id = $1
	`
	org := &domain.Tenant{}
	var role *string
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.DBName,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.MembersCount,
		&role,
	)
	if err != nil {
		return nil, err
	}
	if role != nil {
		org.CurrentUserRole = *role
	}
	return org, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, status string) error {
	query := `UPDATE tenants SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, time.Now().UTC(), tenantID)
	return err
}

func (r *PostgresRepository) AddMembership(ctx context.Context, userID, tenantID, role string) error {
	query := `INSERT INTO tenant_memberships (user_id, tenant_id, role) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, userID, tenantID, role)
	return err
}

func (r *PostgresRepository) ListAll(ctx context.Context) ([]domain.Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_name, slug, db_name, status, created_at, updated_at
		FROM tenants
		ORDER BY tenant_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Tenant, 0)
	for rows.Next() {
		var org domain.Tenant
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.DBName, &org.Status, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, org)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}
