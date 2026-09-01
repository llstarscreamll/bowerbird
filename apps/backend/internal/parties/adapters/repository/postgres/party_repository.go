package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	"github.com/bowerbird/internal/platform/database"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PartyRepository struct {
	registry *database.Registry
}

func NewPartyRepository(registry *database.Registry) *PartyRepository {
	return &PartyRepository{registry: registry}
}

var _ ports.PartyRepository = (*PartyRepository)(nil)

func (r *PartyRepository) Create(ctx context.Context, party domain.Party) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO parties (id, tax_id, name, roles, status, creation_source, created_at, updated_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8)
	`, party.ID, party.TaxID, party.Name, party.Roles, party.Status, party.CreationSource, party.CreatedAt, party.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
		}
		return fmt.Errorf("create party: %w", err)
	}
	return nil
}

func (r *PartyRepository) Update(ctx context.Context, party domain.Party) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	tag, err := pool.Exec(ctx, `
		UPDATE parties
		SET tax_id = NULLIF($2, ''), name = $3, roles = $4, status = $5, updated_at = $6
		WHERE id = $1
	`, party.ID, party.TaxID, party.Name, party.Roles, party.Status, party.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
		}
		return fmt.Errorf("update party: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "party not found")
	}
	return nil
}

func (r *PartyRepository) GetByID(ctx context.Context, id string) (*domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	row := pool.QueryRow(ctx, `
		SELECT id, COALESCE(tax_id, ''), name, roles, status, creation_source, created_at, updated_at
		FROM parties WHERE id = $1
	`, id)
	return scanParty(row)
}

func (r *PartyRepository) GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	row := pool.QueryRow(ctx, `
		SELECT id, COALESCE(tax_id, ''), name, roles, status, creation_source, created_at, updated_at
		FROM parties WHERE tax_id = $1
	`, taxID)
	return scanParty(row)
}

func (r *PartyRepository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	query := `
		SELECT id, COALESCE(tax_id, ''), name, roles, status, creation_source, created_at, updated_at
		FROM parties
		WHERE 1=1
	`
	args := []any{}
	argN := 1
	if role := strings.TrimSpace(filter.Role); role != "" {
		query += fmt.Sprintf(` AND $%d = ANY(roles)`, argN)
		args = append(args, role)
		argN++
	}
	if source := strings.TrimSpace(filter.CreationSource); source != "" {
		query += fmt.Sprintf(` AND creation_source = $%d`, argN)
		args = append(args, source)
		argN++
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR COALESCE(tax_id, '') ILIKE $%d)`, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}
	query += ` ORDER BY name ASC`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list parties: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Party, 0)
	for rows.Next() {
		var party domain.Party
		if err := rows.Scan(&party.ID, &party.TaxID, &party.Name, &party.Roles, &party.Status, &party.CreationSource, &party.CreatedAt, &party.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan party: %w", err)
		}
		out = append(out, party)
	}
	return out, rows.Err()
}

func scanParty(row pgx.Row) (*domain.Party, error) {
	var party domain.Party
	err := row.Scan(&party.ID, &party.TaxID, &party.Name, &party.Roles, &party.Status, &party.CreationSource, &party.CreatedAt, &party.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan party: %w", err)
	}
	return &party, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
