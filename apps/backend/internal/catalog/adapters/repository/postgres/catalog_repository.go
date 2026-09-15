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
	_ ports.ImportRepository       = (*CatalogRepository)(nil)
)

const itemSelectCols = `id, name, kind, status, creation_source, COALESCE(internal_code, ''), created_at, updated_at`

type itemScanner interface {
	Scan(dest ...any) error
}

func scanItem(s itemScanner) (domain.Item, error) {
	var item domain.Item
	err := s.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.CreationSource, &item.InternalCode, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *CatalogRepository) CreateItem(ctx context.Context, item domain.Item) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, kind, status, creation_source, internal_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, item.ID, item.Name, item.Kind, item.Status, item.CreationSource, nullIfEmpty(item.InternalCode), item.CreatedAt, item.UpdatedAt)
	if err != nil {
		if isInternalCodeConflict(err) {
			return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
		}
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
		}
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
		UPDATE catalog_items SET name=$2, kind=$3, status=$4, internal_code=$5, updated_at=$6 WHERE id=$1
	`, item.ID, item.Name, item.Kind, item.Status, nullIfEmpty(item.InternalCode), item.UpdatedAt)
	if err != nil {
		if isInternalCodeConflict(err) {
			return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
		}
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
	item, err := scanItem(pool.QueryRow(ctx, `SELECT `+itemSelectCols+` FROM catalog_items WHERE id=$1`, id))
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
	items, err := r.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out[item.ID] = item.Name
	}
	return out, nil
}

func (r *CatalogRepository) GetItemsByIDs(ctx context.Context, ids []string) ([]domain.Item, error) {
	unique := uniqueIDs(ids)
	if len(unique) == 0 {
		return nil, nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT `+itemSelectCols+` FROM catalog_items WHERE id = ANY($1)`, unique)
	if err != nil {
		return nil, fmt.Errorf("get catalog items by ids: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0, len(unique))
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) ListItems(ctx context.Context, filter ports.ItemListFilter) (ports.ItemListPage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return ports.ItemListPage{}, fmt.Errorf("get tenant db pool: %w", err)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := `SELECT ` + itemSelectCols + ` FROM catalog_items WHERE 1=1`
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
	if source := strings.TrimSpace(filter.CreationSource); source != "" {
		query += fmt.Sprintf(` AND creation_source=$%d`, n)
		args = append(args, source)
		n++
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR internal_code ILIKE $%d OR EXISTS (
			SELECT 1 FROM catalog_item_aliases a WHERE a.item_id = catalog_items.id AND a.value ILIKE $%d
		))`, n, n, n)
		args = append(args, "%"+search+"%")
		n++
	}
	if filter.AfterName != "" && filter.AfterID != "" {
		query += fmt.Sprintf(` AND (name, id) > ($%d, $%d)`, n, n+1)
		args = append(args, filter.AfterName, filter.AfterID)
		n += 2
	}
	query += fmt.Sprintf(` ORDER BY name ASC, id ASC LIMIT $%d`, n)
	args = append(args, limit+1)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return ports.ItemListPage{}, fmt.Errorf("list catalog items: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0, limit)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return ports.ItemListPage{}, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return ports.ItemListPage{}, err
	}
	page := ports.ItemListPage{Items: out}
	if len(out) > limit {
		page.HasMore = true
		page.Items = out[:limit]
	}
	return page, nil
}

func (r *CatalogRepository) GetItemsByInternalCodes(ctx context.Context, codes []string) ([]domain.Item, error) {
	unique := uniqueIDs(codes)
	if len(unique) == 0 {
		return nil, nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT `+itemSelectCols+` FROM catalog_items WHERE internal_code = ANY($1)`, unique)
	if err != nil {
		return nil, fmt.Errorf("get catalog items by internal codes: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0, len(unique))
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) CreateItems(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	return insertItems(ctx, pool, items)
}

func (r *CatalogRepository) UpdateItems(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	return updateItems(ctx, pool, items)
}

func (r *CatalogRepository) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT `+itemSelectCols+`
		FROM catalog_items
		WHERE lower(regexp_replace(btrim(name), '\s+', ' ', 'g')) = $1
	`, normalizedDesc)
	if err != nil {
		return nil, fmt.Errorf("find by description: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
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
		INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, aliasSource(alias), alias.CreatedAt, alias.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return r.aliasConflict(ctx, alias)
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
		SELECT id, item_id, scheme, party_id, value, COALESCE(source, 'invoice'), created_at, updated_at
		FROM catalog_item_aliases
		WHERE scheme = $1 AND COALESCE(party_id, '') = COALESCE(NULLIF($2, ''), '') AND value = $3
	`, scheme, partyID, value).Scan(
		&alias.ID, &alias.ItemID, &alias.Scheme, &alias.PartyID, &alias.Value, &alias.Source, &alias.CreatedAt, &alias.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find alias: %w", err)
	}
	return &alias, nil
}

func (r *CatalogRepository) CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error {
	return r.CreateItemWithAliases(ctx, item, []domain.Alias{alias})
}

func (r *CatalogRepository) CreateItemWithAliases(ctx context.Context, item domain.Item, aliases []domain.Alias) error {
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
		INSERT INTO catalog_items (id, name, kind, status, creation_source, internal_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, item.ID, item.Name, item.Kind, item.Status, item.CreationSource, nullIfEmpty(item.InternalCode), item.CreatedAt, item.UpdatedAt); err != nil {
		if isInternalCodeConflict(err) {
			return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
		}
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
		}
		return fmt.Errorf("create catalog item: %w", err)
	}
	for _, alias := range aliases {
		if _, err := tx.Exec(ctx, `
			INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, source, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, aliasSource(alias), alias.CreatedAt, alias.UpdatedAt); err != nil {
			if isUniqueViolation(err) {
				return r.aliasConflict(ctx, alias)
			}
			return fmt.Errorf("create alias: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create item+aliases: %w", err)
	}
	return nil
}

func (r *CatalogRepository) ListAliasesByItemID(ctx context.Context, itemID string) ([]domain.Alias, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, item_id, scheme, party_id, value, COALESCE(source, 'invoice'), created_at, updated_at
		FROM catalog_item_aliases
		WHERE item_id = $1
		ORDER BY scheme, value
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Alias, 0)
	for rows.Next() {
		var alias domain.Alias
		if err := rows.Scan(&alias.ID, &alias.ItemID, &alias.Scheme, &alias.PartyID, &alias.Value, &alias.Source, &alias.CreatedAt, &alias.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, alias)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) DeleteAlias(ctx context.Context, itemID, aliasID string) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tag, err := pool.Exec(ctx, `DELETE FROM catalog_item_aliases WHERE id = $1 AND item_id = $2`, aliasID, itemID)
	if err != nil {
		return fmt.Errorf("delete alias: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "alias not found")
	}
	return nil
}

func (r *CatalogRepository) RememberDecision(ctx context.Context, aliases []domain.Alias, memory domain.MatchMemory) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, alias := range aliases {
		party := ""
		if alias.PartyID != nil {
			party = *alias.PartyID
		}
		var existing domain.Alias
		err := tx.QueryRow(ctx, `
			SELECT id, item_id, scheme, party_id, value, COALESCE(source, 'invoice'), created_at, updated_at
			FROM catalog_item_aliases
			WHERE scheme = $1 AND COALESCE(party_id, '') = COALESCE(NULLIF($2, ''), '') AND value = $3
		`, alias.Scheme, party, alias.Value).Scan(
			&existing.ID, &existing.ItemID, &existing.Scheme, &existing.PartyID, &existing.Value, &existing.Source, &existing.CreatedAt, &existing.UpdatedAt,
		)
		if err == nil {
			if !existing.PointsTo(alias.ItemID) {
				return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item").
					WithMeta("item_id", existing.ItemID)
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find alias: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO catalog_item_aliases (id, item_id, scheme, party_id, value, source, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, alias.ID, alias.ItemID, alias.Scheme, alias.PartyID, alias.Value, aliasSource(alias), alias.CreatedAt, alias.UpdatedAt); err != nil {
			if isUniqueViolation(err) {
				return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
			}
			return fmt.Errorf("create alias: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO catalog_match_memories (
			id, evidence_key, party_id, item_code, description_fingerprint,
			evidence_kind, item_id, action, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (evidence_key) DO UPDATE SET
			party_id = EXCLUDED.party_id,
			item_code = EXCLUDED.item_code,
			description_fingerprint = EXCLUDED.description_fingerprint,
			evidence_kind = EXCLUDED.evidence_kind,
			item_id = EXCLUDED.item_id,
			action = EXCLUDED.action,
			updated_at = EXCLUDED.updated_at
	`, memory.ID, memory.EvidenceKey, memory.PartyID, nullIfEmpty(memory.ItemCode), nullIfEmpty(memory.DescriptionFingerprint),
		memory.EvidenceKind, memory.ItemID, memory.Action, memory.CreatedAt, memory.UpdatedAt); err != nil {
		return fmt.Errorf("upsert match memory: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit remember decision: %w", err)
	}
	return nil
}

func aliasSource(alias domain.Alias) string {
	if strings.TrimSpace(alias.Source) == "" {
		return domain.AliasSourceInvoice
	}
	return alias.Source
}

func (r *CatalogRepository) aliasConflict(ctx context.Context, alias domain.Alias) error {
	party := ""
	if alias.PartyID != nil {
		party = *alias.PartyID
	}
	existing, err := r.FindBySchemePartyValue(ctx, alias.Scheme, party, alias.Value)
	if err == nil && existing != nil {
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item").
			WithMeta("item_id", existing.ItemID)
	}
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
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

func uniqueIDs(ids []string) []string {
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
	return unique
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isInternalCodeConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_catalog_items_internal_code"
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
