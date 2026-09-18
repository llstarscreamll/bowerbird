package postgres

import (
	"context"
	"fmt"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
)

func (r *CatalogRepository) ApplyMerge(ctx context.Context, in ports.MergePersistence) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin merge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE catalog_items
		SET name=$2, kind=$3, status=$4, internal_code=$5, merged_into_id=$6, updated_at=$7
		WHERE id=$1
	`, in.Survivor.ID, in.Survivor.Name, in.Survivor.Kind, in.Survivor.Status,
		nullIfEmpty(in.Survivor.InternalCode), nullIfEmpty(in.Survivor.MergedIntoID), in.Survivor.UpdatedAt); err != nil {
		if isInternalCodeConflict(err) {
			return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
		}
		return fmt.Errorf("update survivor: %w", err)
	}

	fromIDs := make([]string, 0, len(in.Merged))
	for _, item := range in.Merged {
		fromIDs = append(fromIDs, item.ID)
		if _, err := tx.Exec(ctx, `
			UPDATE catalog_items
			SET status=$2, merged_into_id=$3, internal_code=NULL, updated_at=$4
			WHERE id=$1
		`, item.ID, item.Status, nullIfEmpty(item.MergedIntoID), item.UpdatedAt); err != nil {
			return fmt.Errorf("update merged item: %w", err)
		}
	}

	if len(in.DeleteAliasIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM catalog_item_aliases WHERE id = ANY($1)`, in.DeleteAliasIDs); err != nil {
			return fmt.Errorf("delete colliding aliases: %w", err)
		}
	}
	for _, alias := range in.ReassignAliases {
		if _, err := tx.Exec(ctx, `
			UPDATE catalog_item_aliases SET item_id=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$1
		`, alias.ID, alias.ItemID); err != nil {
			if isUniqueViolation(err) {
				return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item").
					WithMeta("item_id", in.Survivor.ID)
			}
			return fmt.Errorf("reassign alias: %w", err)
		}
	}

	if len(fromIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM catalog_match_memories src
			USING catalog_match_memories dst
			WHERE src.item_id = ANY($1)
			  AND dst.item_id = $2
			  AND src.evidence_key = dst.evidence_key
			  AND (dst.action = 'link' OR src.action <> 'link')
		`, fromIDs, in.Survivor.ID); err != nil {
			return fmt.Errorf("drop colliding source memories: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM catalog_match_memories dst
			USING catalog_match_memories src
			WHERE src.item_id = ANY($1)
			  AND dst.item_id = $2
			  AND src.evidence_key = dst.evidence_key
			  AND src.action = 'link' AND dst.action <> 'link'
		`, fromIDs, in.Survivor.ID); err != nil {
			return fmt.Errorf("drop colliding survivor never_match: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE catalog_match_memories SET item_id=$1, updated_at=$2 WHERE item_id = ANY($3)
		`, in.Survivor.ID, in.Survivor.UpdatedAt, fromIDs); err != nil {
			return fmt.Errorf("reassign match memories: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit merge: %w", err)
	}
	return nil
}

func (r *CatalogRepository) DescriptionDuplicateGroups(ctx context.Context) ([][]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT `+itemSelectCols+` FROM catalog_items WHERE status <> 'merged'`)
	if err != nil {
		return nil, fmt.Errorf("list items for description groups: %w", err)
	}
	defer rows.Close()
	byNorm := map[string][]domain.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		norm := domain.NormalizeDescription(item.Name)
		if norm == "" {
			continue
		}
		byNorm[norm] = append(byNorm[norm], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([][]domain.Item, 0)
	for _, group := range byNorm {
		if len(group) > 1 {
			out = append(out, group)
		}
	}
	return out, nil
}

func (r *CatalogRepository) CrossPartySKUGroups(ctx context.Context) ([][]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT a.value, a.item_id
		FROM catalog_item_aliases a
		JOIN catalog_items i ON i.id = a.item_id
		WHERE a.scheme = $1 AND a.party_id IS NOT NULL AND i.status <> 'merged'
	`, domain.AliasSchemeSupplierSKU)
	if err != nil {
		return nil, fmt.Errorf("cross-party sku aliases: %w", err)
	}
	defer rows.Close()
	type hit struct {
		value  string
		itemID string
	}
	var hits []hit
	itemIDs := map[string]struct{}{}
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.value, &h.itemID); err != nil {
			return nil, err
		}
		hits = append(hits, h)
		itemIDs[h.itemID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(itemIDs))
	for id := range itemIDs {
		ids = append(ids, id)
	}
	items, err := r.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.Item{}
	for _, item := range items {
		byID[item.ID] = item
	}
	partiesByValue := map[string]map[string]struct{}{}
	itemsByValue := map[string]map[string]struct{}{}
	for _, h := range hits {
		if itemsByValue[h.value] == nil {
			itemsByValue[h.value] = map[string]struct{}{}
			partiesByValue[h.value] = map[string]struct{}{}
		}
		itemsByValue[h.value][h.itemID] = struct{}{}
	}
	rows2, err := pool.Query(ctx, `
		SELECT a.value, a.party_id, a.item_id
		FROM catalog_item_aliases a
		JOIN catalog_items i ON i.id = a.item_id
		WHERE a.scheme = $1 AND a.party_id IS NOT NULL AND i.status <> 'merged'
	`, domain.AliasSchemeSupplierSKU)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var value, party, itemID string
		if err := rows2.Scan(&value, &party, &itemID); err != nil {
			return nil, err
		}
		if partiesByValue[value] == nil {
			partiesByValue[value] = map[string]struct{}{}
			itemsByValue[value] = map[string]struct{}{}
		}
		partiesByValue[value][party] = struct{}{}
		itemsByValue[value][itemID] = struct{}{}
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}
	out := make([][]domain.Item, 0)
	for value, itemsSet := range itemsByValue {
		if len(itemsSet) < 2 || len(partiesByValue[value]) < 2 {
			continue
		}
		group := make([]domain.Item, 0, len(itemsSet))
		for id := range itemsSet {
			if item, ok := byID[id]; ok {
				group = append(group, item)
			}
		}
		if len(group) > 1 {
			out = append(out, group)
		}
	}
	return out, nil
}

func (r *CatalogRepository) UpsertNotDuplicatePairs(ctx context.Context, pairs []domain.NotDuplicatePair) error {
	if len(pairs) == 0 {
		return nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	for _, pair := range pairs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO catalog_item_not_duplicates (item_lo, item_hi)
			VALUES ($1, $2)
			ON CONFLICT (item_lo, item_hi) DO NOTHING
		`, pair.Left, pair.Right); err != nil {
			return fmt.Errorf("upsert not-duplicate pair: %w", err)
		}
	}
	return nil
}

func (r *CatalogRepository) ListNotDuplicatePairs(ctx context.Context) ([]domain.NotDuplicatePair, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT item_lo, item_hi FROM catalog_item_not_duplicates`)
	if err != nil {
		return nil, fmt.Errorf("list not-duplicate pairs: %w", err)
	}
	defer rows.Close()
	out := make([]domain.NotDuplicatePair, 0)
	for rows.Next() {
		var pair domain.NotDuplicatePair
		if err := rows.Scan(&pair.Left, &pair.Right); err != nil {
			return nil, err
		}
		out = append(out, pair)
	}
	return out, rows.Err()
}
