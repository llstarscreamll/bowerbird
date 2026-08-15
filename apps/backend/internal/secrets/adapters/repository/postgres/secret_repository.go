package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bowerbird/internal/platform/database"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SecretRepository struct {
	registry *database.Registry
}

func NewSecretRepository(registry *database.Registry) *SecretRepository {
	return &SecretRepository{registry: registry}
}

var _ ports.SecretRepository = (*SecretRepository)(nil)

func (r *SecretRepository) List(ctx context.Context, purpose string) ([]domain.Secret, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	query := `
		SELECT id, purpose, kind, label, COALESCE(description, ''), version, key_id,
		       last_used_at, created_by, updated_by, created_at, updated_at
		FROM secrets
	`
	args := []any{}
	if purpose != "" {
		query += ` WHERE purpose = $1`
		args = append(args, purpose)
	}
	query += ` ORDER BY purpose ASC, label ASC`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	return scanSecretMetadataRows(rows)
}

func (r *SecretRepository) GetByID(ctx context.Context, id string) (*domain.Secret, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	row := pool.QueryRow(ctx, `
		SELECT id, purpose, kind, label, COALESCE(description, ''), ciphertext, version, key_id,
		       last_used_at, created_by, updated_by, created_at, updated_at
		FROM secrets
		WHERE id = $1
	`, id)

	secret, err := scanSecretRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	return secret, nil
}

func (r *SecretRepository) Create(ctx context.Context, secret domain.Secret) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO secrets (
			id, purpose, kind, label, description, ciphertext, version, key_id,
			last_used_at, created_by, updated_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8,
			$9, $10, $11, $12, $13
		)
	`,
		secret.ID, secret.Purpose, secret.Kind, secret.Label, secret.Description, secret.Ciphertext, secret.Version, secret.KeyID,
		secret.LastUsedAt, secret.CreatedBy, secret.UpdatedBy, secret.CreatedAt, secret.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a secret with this purpose and label already exists")
		}
		return fmt.Errorf("create secret: %w", err)
	}
	return nil
}

func (r *SecretRepository) Update(ctx context.Context, secret domain.Secret) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	cmdTag, err := pool.Exec(ctx, `
		UPDATE secrets
		SET label = $2,
		    description = NULLIF($3, ''),
		    ciphertext = $4,
		    version = $5,
		    updated_by = $6,
		    updated_at = $7
		WHERE id = $1
	`, secret.ID, secret.Label, secret.Description, secret.Ciphertext, secret.Version, secret.UpdatedBy, secret.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a secret with this purpose and label already exists")
		}
		return fmt.Errorf("update secret: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "secret not found")
	}
	return nil
}

func (r *SecretRepository) Delete(ctx context.Context, id string) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	cmdTag, err := pool.Exec(ctx, `DELETE FROM secrets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "secret not found")
	}
	return nil
}

func (r *SecretRepository) ListCiphertextsByPurpose(ctx context.Context, purpose string) ([]domain.Secret, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT id, purpose, kind, label, COALESCE(description, ''), ciphertext, version, key_id,
		       last_used_at, created_by, updated_by, created_at, updated_at
		FROM secrets
		WHERE purpose = $1
		ORDER BY last_used_at DESC NULLS LAST, created_at ASC
	`, purpose)
	if err != nil {
		return nil, fmt.Errorf("list secrets by purpose: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Secret, 0)
	for rows.Next() {
		secret, err := scanSecretFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *secret)
	}
	return out, rows.Err()
}

func (r *SecretRepository) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	_, err = pool.Exec(ctx, `
		UPDATE secrets
		SET last_used_at = $2, updated_at = $2
		WHERE id = $1
	`, id, usedAt)
	if err != nil {
		return fmt.Errorf("mark secret used: %w", err)
	}
	return nil
}

func (r *SecretRepository) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO secret_audit_events (id, secret_id, purpose, action, actor_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.ID, event.SecretID, event.Purpose, event.Action, event.ActorUserID, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("append secret audit: %w", err)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSecretRow(row scannable) (*domain.Secret, error) {
	var secret domain.Secret
	err := row.Scan(
		&secret.ID, &secret.Purpose, &secret.Kind, &secret.Label, &secret.Description, &secret.Ciphertext, &secret.Version, &secret.KeyID,
		&secret.LastUsedAt, &secret.CreatedBy, &secret.UpdatedBy, &secret.CreatedAt, &secret.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func scanSecretFromRows(rows pgx.Rows) (*domain.Secret, error) {
	return scanSecretRow(rows)
}

func scanSecretMetadataRows(rows pgx.Rows) ([]domain.Secret, error) {
	out := make([]domain.Secret, 0)
	for rows.Next() {
		var secret domain.Secret
		err := rows.Scan(
			&secret.ID, &secret.Purpose, &secret.Kind, &secret.Label, &secret.Description, &secret.Version, &secret.KeyID,
			&secret.LastUsedAt, &secret.CreatedBy, &secret.UpdatedBy, &secret.CreatedAt, &secret.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		out = append(out, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "secrets_purpose_label_unique")
}
