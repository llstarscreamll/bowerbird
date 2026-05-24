package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/money-path/bowerbird/apps/backend/internal/identity/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
)

type PostgresRepository struct {
	controlDB *pgxpool.Pool
	registry  *database.Registry
}

func NewPostgresRepository(controlDB *pgxpool.Pool, registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{
		controlDB: controlDB,
		registry:  registry,
	}
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, first_name, last_name, picture_url, created_at, updated_at, deleted_at FROM users WHERE email = $1 AND deleted_at IS NULL`
	var user domain.User
	var pictureURL *string
	err := r.controlDB.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &pictureURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if pictureURL != nil {
		user.PictureURL = *pictureURL
	}
	return &user, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, first_name, last_name, picture_url, created_at, updated_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`
	var user domain.User
	var pictureURL *string
	err := r.controlDB.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &pictureURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if pictureURL != nil {
		user.PictureURL = *pictureURL
	}
	return &user, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, first_name, last_name, picture_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var pictureURL *string
	if user.PictureURL != "" {
		pictureURL = &user.PictureURL
	}
	err := r.controlDB.QueryRow(ctx, query, user.Email, user.FirstName, user.LastName, pictureURL, user.CreatedAt, user.UpdatedAt).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindUserIdentity(ctx context.Context, provider, providerID string) (*domain.UserIdentity, error) {
	query := `SELECT id, user_id, provider, provider_id, created_at FROM user_identities WHERE provider = $1 AND provider_id = $2`
	var identity domain.UserIdentity
	err := r.controlDB.QueryRow(ctx, query, provider, providerID).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderID, &identity.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user identity: %w", err)
	}
	return &identity, nil
}

func (r *PostgresRepository) FindUserIdentityByProvider(ctx context.Context, userID, provider string) (*domain.UserIdentity, error) {
	query := `SELECT id, user_id, provider, provider_id, created_at FROM user_identities WHERE user_id = $1 AND provider = $2`
	var identity domain.UserIdentity
	err := r.controlDB.QueryRow(ctx, query, userID, provider).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderID, &identity.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user identity by provider: %w", err)
	}
	return &identity, nil
}

func (r *PostgresRepository) CreateUserIdentity(ctx context.Context, identity *domain.UserIdentity) error {
	query := `INSERT INTO user_identities (user_id, provider, provider_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	err := r.controlDB.QueryRow(ctx, query, identity.UserID, identity.Provider, identity.ProviderID, identity.CreatedAt).Scan(&identity.ID)
	if err != nil {
		return fmt.Errorf("failed to create user identity: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindTenantMemberships(ctx context.Context, userID string) ([]*domain.TenantMembership, error) {
	query := `
		SELECT m.user_id, m.tenant_id, m.role, m.created_at, m.deleted_at 
		FROM tenant_memberships m
		JOIN tenants t ON m.tenant_id = t.id
		WHERE m.user_id = $1 AND m.deleted_at IS NULL AND t.status = 'active'
	`
	rows, err := r.controlDB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*domain.TenantMembership
	for rows.Next() {
		var m domain.TenantMembership
		if err := rows.Scan(&m.UserID, &m.TenantID, &m.Role, &m.CreatedAt, &m.DeletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan tenant membership: %w", err)
		}
		memberships = append(memberships, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant memberships: %w", err)
	}
	return memberships, nil
}

func (r *PostgresRepository) AddTenantMembership(ctx context.Context, membership *domain.TenantMembership) error {
	query := `INSERT INTO tenant_memberships (user_id, tenant_id, role, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.controlDB.Exec(ctx, query, membership.UserID, membership.TenantID, membership.Role, membership.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add tenant membership: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RemoveTenantMembership(ctx context.Context, userID, tenantID string) error {
	query := `UPDATE tenant_memberships SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	_, err := r.controlDB.Exec(ctx, query, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to remove tenant membership: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP, email = CONCAT(email, '-deleted-', gen_random_uuid()), first_name = 'Deleted', last_name = 'User' WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.controlDB.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	// Also soft delete all memberships
	memQuery := `UPDATE tenant_memberships SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND deleted_at IS NULL`
	_, err = r.controlDB.Exec(ctx, memQuery, userID)
	return err
}

func (r *PostgresRepository) SoftDeleteTenantUserProfile(ctx context.Context, tenantDBName string, userID string) error {
	pool, err := r.registry.GetPoolByDBName(ctx, tenantDBName)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP, email = CONCAT(email, '-deleted-', gen_random_uuid()), first_name = 'Deleted', last_name = 'User', status = 'inactive' WHERE id = $1 AND deleted_at IS NULL`
	_, err = pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete tenant user profile: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetTenantDBName(ctx context.Context, tenantID string) (string, error) {
	query := `SELECT db_name FROM tenants WHERE id = $1`
	var dbName string
	err := r.controlDB.QueryRow(ctx, query, tenantID).Scan(&dbName)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant db_name: %w", err)
	}
	return dbName, nil
}

// CreateTenantUserProfile inserts the profile into the tenant's database
func (r *PostgresRepository) CreateTenantUserProfile(ctx context.Context, tenantDBName string, profile *domain.TenantUserProfile) error {
	// Wait, the interface says CreateTenantUserProfile(ctx context.Context, tenantDBName string, profile *TenantUserProfile) error
	// We can't use registry.GetPool(ctx) because we might not have the slug in the context, but we can do a workaround
	// Let's assume there's a function we'll add to registry: GetPoolByDBName
	pool, err := r.registry.GetPoolByDBName(ctx, tenantDBName)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `INSERT INTO users (id, first_name, last_name, avatar_url, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = pool.Exec(ctx, query, profile.ID, profile.FirstName, profile.LastName, profile.AvatarURL, profile.Status, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create tenant user profile: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateTenantUserProfile(ctx context.Context, tenantDBName string, profile *domain.TenantUserProfile) error {
	pool, err := r.registry.GetPoolByDBName(ctx, tenantDBName)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `UPDATE users SET first_name = $1, last_name = $2, avatar_url = $3, status = $4, updated_at = $5 WHERE id = $6`
	_, err = pool.Exec(ctx, query, profile.FirstName, profile.LastName, profile.AvatarURL, profile.Status, profile.UpdatedAt, profile.ID)
	if err != nil {
		return fmt.Errorf("failed to update tenant user profile: %w", err)
	}
	return nil
}
