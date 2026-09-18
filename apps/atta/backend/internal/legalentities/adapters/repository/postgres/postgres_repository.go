package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/atta/internal/legalentities/application/ports"
	"github.com/atta/internal/legalentities/domain"
	"github.com/atta/internal/platform/database"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	registry *database.Registry
}

func NewRepository(registry *database.Registry) *Repository {
	return &Repository{registry: registry}
}

var _ ports.LegalEntityRepository = (*Repository)(nil)

func (r *Repository) Count(ctx context.Context) (int, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return 0, fmt.Errorf("get tenant db pool: %w", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM legal_entities`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count legal entities: %w", err)
	}
	return count, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.LegalEntity, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, tax_id, scheme_id, legal_name, created_at, updated_at
		FROM legal_entities
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list legal entities: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LegalEntity, 0)
	for rows.Next() {
		var entity domain.LegalEntity
		if err := rows.Scan(&entity.ID, &entity.TaxID, &entity.SchemeID, &entity.LegalName, &entity.CreatedAt, &entity.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan legal entity: %w", err)
		}
		out = append(out, entity)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.LegalEntity, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	row := pool.QueryRow(ctx, `
		SELECT id, tax_id, scheme_id, legal_name, created_at, updated_at
		FROM legal_entities WHERE id = $1
	`, id)
	var entity domain.LegalEntity
	err = row.Scan(&entity.ID, &entity.TaxID, &entity.SchemeID, &entity.LegalName, &entity.CreatedAt, &entity.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan legal entity: %w", err)
	}
	return &entity, nil
}

func (r *Repository) Create(ctx context.Context, entity domain.LegalEntity) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO legal_entities (id, tax_id, scheme_id, legal_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, entity.ID, entity.TaxID, entity.SchemeID, entity.LegalName, entity.CreatedAt, entity.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a legal entity with this tax id already exists")
		}
		return fmt.Errorf("create legal entity: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, entity domain.LegalEntity) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tag, err := pool.Exec(ctx, `
		UPDATE legal_entities
		SET tax_id = $2, scheme_id = $3, legal_name = $4, updated_at = $5
		WHERE id = $1
	`, entity.ID, entity.TaxID, entity.SchemeID, entity.LegalName, entity.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a legal entity with this tax id already exists")
		}
		return fmt.Errorf("update legal entity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "legal entity not found")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
