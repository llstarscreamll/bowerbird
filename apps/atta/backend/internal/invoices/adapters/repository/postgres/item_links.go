package postgres

import (
	"context"
	"fmt"

	"github.com/atta/internal/invoices/application/ports"
	"github.com/atta/internal/invoices/domain"
)

func (r *PostgresRepository) RelinkCatalogItems(ctx context.Context, fromIDs []string, toID string) error {
	if len(fromIDs) == 0 || toID == "" {
		return nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin catalog item relink: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE invoice_lines
		SET item_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE item_id = ANY($2)
	`, toID, fromIDs); err != nil {
		return fmt.Errorf("relink invoice lines: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE invoice_lines
		SET suggestions = COALESCE((
			SELECT jsonb_agg(
				CASE
					WHEN elem->>'item_id' = ANY($2::text[])
						THEN jsonb_set(elem, '{item_id}', to_jsonb($1::text))
					ELSE elem
				END
			)
			FROM jsonb_array_elements(suggestions) AS elem
		), '[]'::jsonb),
		    updated_at = CURRENT_TIMESTAMP
		WHERE suggestions IS NOT NULL
		  AND jsonb_typeof(suggestions) = 'array'
		  AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(suggestions) e
			WHERE e->>'item_id' = ANY($2::text[])
		  )
	`, toID, fromIDs); err != nil {
		return fmt.Errorf("rewrite line suggestions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit catalog item relink: %w", err)
	}
	return nil
}

func (r *PostgresRepository) HardConflictItemPairs(ctx context.Context) ([]ports.ItemIDPair, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT
			LEAST(s1.elem->>'item_id', s2.elem->>'item_id'),
			GREATEST(s1.elem->>'item_id', s2.elem->>'item_id')
		FROM invoice_lines l
		CROSS JOIN LATERAL jsonb_array_elements(l.suggestions) AS s1(elem)
		CROSS JOIN LATERAL jsonb_array_elements(l.suggestions) AS s2(elem)
		WHERE s1.elem->>'reason' = $1
		  AND s2.elem->>'reason' = $1
		  AND s1.elem->>'item_id' < s2.elem->>'item_id'
	`, domain.SuggestionReasonHardConflict)
	if err != nil {
		return nil, fmt.Errorf("hard conflict pairs: %w", err)
	}
	defer rows.Close()
	out := make([]ports.ItemIDPair, 0)
	for rows.Next() {
		var pair ports.ItemIDPair
		if err := rows.Scan(&pair.Left, &pair.Right); err != nil {
			return nil, err
		}
		if pair.Left == "" || pair.Right == "" {
			continue
		}
		out = append(out, pair)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CountLinesByItemIDs(ctx context.Context, itemIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT item_id, COUNT(*) FROM invoice_lines WHERE item_id = ANY($1) GROUP BY item_id
	`, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("count lines by item: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
