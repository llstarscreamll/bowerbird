package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	"github.com/bowerbird/internal/platform/database"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type CatalogRepository struct {
	registry *database.Registry
}

func NewCatalogRepository(registry *database.Registry) *CatalogRepository {
	return &CatalogRepository{registry: registry}
}

var (
	_ ports.ItemRepository         = (*CatalogRepository)(nil)
	_ ports.AliasRepository        = (*CatalogRepository)(nil)
	_ ports.CatalogWriteRepository = (*CatalogRepository)(nil)
	_ ports.MatchMemoryRepository  = (*CatalogRepository)(nil)
)

func (r *CatalogRepository) CreateItem(ctx context.Context, item domain.Item) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, kind, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.Name, item.Kind, item.Status, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create catalog item: %w", err)
	}
	return nil
}

func (r *CatalogRepository) UpdateItem(ctx context.Context, item domain.Item) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tag, err := pool.Exec(ctx, `
		UPDATE catalog_items SET name=$2, kind=$3, status=$4, updated_at=$5 WHERE id=$1
	`, item.ID, item.Name, item.Kind, item.Status, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update catalog item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return nil
}

func (r *CatalogRepository) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	var item domain.Item
	err = pool.QueryRow(ctx, `
		SELECT id, name, kind, status, created_at, updated_at FROM catalog_items WHERE id=$1
	`, id).Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog item: %w", err)
	}
	return &item, nil
}

func (r *CatalogRepository) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `SELECT id, name FROM catalog_items WHERE id = ANY($1)`, unique)
	if err != nil {
		return nil, fmt.Errorf("get catalog item names: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

func (r *CatalogRepository) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	query := `SELECT id, name, kind, status, created_at, updated_at FROM catalog_items WHERE 1=1`
	args := []any{}
	n := 1
	if filter.Kind != "" {
		query += fmt.Sprintf(` AND kind=$%d`, n)
		args = append(args, filter.Kind)
		n++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(` AND status=$%d`, n)
		args = append(args, filter.Status)
		n++
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR EXISTS (
			SELECT 1 FROM catalog_item_aliases a WHERE a.item_id = catalog_items.id AND a.value ILIKE $%d
		))`, n, n)
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY name ASC`
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog items: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, name, kind, status, created_at, updated_at
		FROM catalog_items
		WHERE lower(regexp_replace(btrim(name), '\s+', ' ', 'g')) = $1
	`, normalizedDesc)
	if err != nil {
		return nil, fmt.Errorf("find by description: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) CreateAlias(ctx context.Context, alias domain.Alias) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, alias.CreatedAt, alias.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
		}
		return fmt.Errorf("create alias: %w", err)
	}
	return nil
}

func (r *CatalogRepository) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	var alias domain.Alias
	err = pool.QueryRow(ctx, `
		SELECT id, item_id, scheme, party_id, value, created_at, updated_at
		FROM catalog_item_aliases
		WHERE scheme = $1 AND COALESCE(party_id, '') = COALESCE(NULLIF($2, ''), '') AND value = $3
	`, scheme, partyID, value).Scan(
		&alias.ID, &alias.ItemID, &alias.Scheme, &alias.PartyID, &alias.Value, &alias.CreatedAt, &alias.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find alias: %w", err)
	}
	return &alias, nil
}

func (r *CatalogRepository) ListInternalSKUsByItemIDs(ctx context.Context, itemIDs []string) (map[string]string, error) {
	out := make(map[string]string, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	unique := make([]string, 0, len(itemIDs))
	seen := make(map[string]struct{}, len(itemIDs))
	for _, id := range itemIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT item_id, value
		FROM catalog_item_aliases
		WHERE scheme = $1 AND item_id = ANY($2)
	`, domain.AliasSchemeInternalSKU, unique)
	if err != nil {
		return nil, fmt.Errorf("list internal skus: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var itemID, value string
		if err := rows.Scan(&itemID, &value); err != nil {
			return nil, err
		}
		out[itemID] = value
	}
	return out, rows.Err()
}

func (r *CatalogRepository) CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO catalog_items (id, name, kind, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.Name, item.Kind, item.Status, item.CreatedAt, item.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
		}
		return fmt.Errorf("create catalog item: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, alias.CreatedAt, alias.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
		}
		return fmt.Errorf("create alias: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create item+alias: %w", err)
	}
	return nil
}

func (r *CatalogRepository) UpdateItemWithOptionalAlias(ctx context.Context, item domain.Item, alias *domain.Alias) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE catalog_items SET name=$2, kind=$3, status=$4, updated_at=$5 WHERE id=$1
	`, item.ID, item.Name, item.Kind, item.Status, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update catalog item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	if alias != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, alias.CreatedAt, alias.UpdatedAt); err != nil {
			if isUniqueViolation(err) {
				return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
			}
			return fmt.Errorf("create alias: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update item: %w", err)
	}
	return nil
}

func (r *CatalogRepository) UpsertMemory(ctx context.Context, memory domain.MatchMemory) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_match_memories (
			id, evidence_key, party_id, item_code, description_fingerprint, evidence_kind,
			item_id, action, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (evidence_key) DO UPDATE SET
			party_id = EXCLUDED.party_id,
			item_code = EXCLUDED.item_code,
			description_fingerprint = EXCLUDED.description_fingerprint,
			evidence_kind = EXCLUDED.evidence_kind,
			item_id = EXCLUDED.item_id,
			action = EXCLUDED.action,
			updated_at = EXCLUDED.updated_at
	`, memory.ID, memory.EvidenceKey, memory.PartyID, nullIfEmpty(memory.ItemCode), nullIfEmpty(memory.DescriptionFingerprint),
		memory.EvidenceKind, memory.ItemID, memory.Action, memory.CreatedAt, memory.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert match memory: %w", err)
	}
	return nil
}

func (r *CatalogRepository) FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	var mem domain.MatchMemory
	err = pool.QueryRow(ctx, `
		SELECT id, evidence_key, party_id, COALESCE(item_code, ''), COALESCE(description_fingerprint, ''),
			evidence_kind, item_id, action, created_at, updated_at
		FROM catalog_match_memories WHERE evidence_key = $1
	`, evidenceKey).Scan(
		&mem.ID, &mem.EvidenceKey, &mem.PartyID, &mem.ItemCode, &mem.DescriptionFingerprint,
		&mem.EvidenceKind, &mem.ItemID, &mem.Action, &mem.CreatedAt, &mem.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find match memory: %w", err)
	}
	return &mem, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
