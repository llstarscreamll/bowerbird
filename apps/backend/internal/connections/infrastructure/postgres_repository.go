package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/money-path/bowerbird/apps/backend/internal/connections/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
)

type PostgresRepository struct {
	registry *database.Registry
}

func NewPostgresRepository(registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{registry: registry}
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Connection, error) {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, owner_user_id, provider, email_address, status, encrypted_credentials, granted_scopes, sharing_policy, raw_data, created_at, updated_at
		FROM connections
		WHERE id = $1
	`
	var c domain.Connection
	var rawData, grantedScopes []byte
	var ownerID *string

	err = conn.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&ownerID,
		&c.Provider,
		&c.ProviderAccountEmail,
		&c.Status,
		&c.EncryptedCredentials,
		&grantedScopes,
		&c.SharingPolicy,
		&rawData,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get connection by id: %w", err)
	}

	if ownerID != nil {
		c.OwnerUserID = *ownerID
	}
	c.RawData = rawData

	if len(grantedScopes) > 0 {
		if err := json.Unmarshal(grantedScopes, &c.GrantedScopes); err != nil {
			return nil, fmt.Errorf("unmarshal granted scopes: %w", err)
		}
	}

	return &c, nil
}

func (r *PostgresRepository) ListAll(ctx context.Context) ([]*domain.Connection, error) {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, owner_user_id, provider, email_address, status, encrypted_credentials, granted_scopes, sharing_policy, raw_data, created_at, updated_at
		FROM connections
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all connections: %w", err)
	}
	defer rows.Close()

	return scanConnections(rows)
}

func (r *PostgresRepository) ListActive(ctx context.Context) ([]*domain.Connection, error) {
	return r.listByStatus(ctx, domain.ConnectionStatusActive)
}

func (r *PostgresRepository) listByStatus(ctx context.Context, status string) ([]*domain.Connection, error) {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, owner_user_id, provider, email_address, status, encrypted_credentials, granted_scopes, sharing_policy, raw_data, created_at, updated_at
		FROM connections
		WHERE status = $1
	`
	rows, err := conn.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("query connections by status: %w", err)
	}
	defer rows.Close()

	return scanConnections(rows)
}

func (r *PostgresRepository) ListByOwner(ctx context.Context, ownerUserID string) ([]*domain.Connection, error) {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, owner_user_id, provider, email_address, status, encrypted_credentials, granted_scopes, sharing_policy, raw_data, created_at, updated_at
		FROM connections
		WHERE owner_user_id = $1
	`
	rows, err := conn.Query(ctx, query, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("query connections by owner: %w", err)
	}
	defer rows.Close()

	return scanConnections(rows)
}

func scanConnections(rows pgx.Rows) ([]*domain.Connection, error) {
	var connections []*domain.Connection
	for rows.Next() {
		var c domain.Connection
		var rawData, grantedScopes []byte
		var ownerID *string

		err := rows.Scan(
			&c.ID,
			&ownerID,
			&c.Provider,
			&c.ProviderAccountEmail,
			&c.Status,
			&c.EncryptedCredentials,
			&grantedScopes,
			&c.SharingPolicy,
			&rawData,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}

		if ownerID != nil {
			c.OwnerUserID = *ownerID
		}
		c.RawData = rawData

		if len(grantedScopes) > 0 {
			if err := json.Unmarshal(grantedScopes, &c.GrantedScopes); err != nil {
				return nil, fmt.Errorf("unmarshal granted scopes: %w", err)
			}
		}

		connections = append(connections, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return connections, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, c *domain.Connection) error {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return err
	}

	grantedScopesJSON, err := json.Marshal(c.GrantedScopes)
	if err != nil {
		return fmt.Errorf("marshal granted scopes: %w", err)
	}

	var ownerID *string
	if c.OwnerUserID != "" {
		ownerID = &c.OwnerUserID
	}

	rawData := c.RawData
	if len(rawData) == 0 {
		rawData = []byte("{}")
	}

	query := `
		INSERT INTO connections (
			id, owner_user_id, provider, email_address, status, encrypted_credentials, granted_scopes, sharing_policy, raw_data, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) ON CONFLICT (provider, email_address) DO UPDATE SET
			owner_user_id = EXCLUDED.owner_user_id,
			provider = EXCLUDED.provider,
			email_address = EXCLUDED.email_address,
			status = EXCLUDED.status,
			encrypted_credentials = EXCLUDED.encrypted_credentials,
			granted_scopes = EXCLUDED.granted_scopes,
			sharing_policy = EXCLUDED.sharing_policy,
			raw_data = EXCLUDED.raw_data,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	err = conn.QueryRow(ctx, query,
		c.ID,
		ownerID,
		c.Provider,
		c.ProviderAccountEmail,
		c.Status,
		c.EncryptedCredentials,
		grantedScopesJSON,
		c.SharingPolicy,
		rawData,
		c.CreatedAt,
		c.UpdatedAt,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("upsert connection: %w", err)
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	conn, err := r.registry.GetPool(ctx)
	if err != nil {
		return err
	}

	query := `DELETE FROM connections WHERE id = $1`
	_, err = conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	return nil
}
