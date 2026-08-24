package postgres

import (
	"context"
	"encoding/json"
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
	_ ports.ItemRepository            = (*CatalogRepository)(nil)
	_ ports.AliasRepository           = (*CatalogRepository)(nil)
	_ ports.MatchMemoryRepository     = (*CatalogRepository)(nil)
	_ ports.InvoiceLineLinkRepository = (*CatalogRepository)(nil)
)

func (r *CatalogRepository) CreateItem(ctx context.Context, item domain.Item) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, kind, status, stockable, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, item.ID, item.Name, item.Kind, item.Status, item.Stockable, item.CreatedAt, item.UpdatedAt)
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
		UPDATE catalog_items SET name=$2, kind=$3, status=$4, stockable=$5, updated_at=$6 WHERE id=$1
	`, item.ID, item.Name, item.Kind, item.Status, item.Stockable, item.UpdatedAt)
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
		SELECT id, name, kind, status, stockable, created_at, updated_at FROM catalog_items WHERE id=$1
	`, id).Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.Stockable, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog item: %w", err)
	}
	return &item, nil
}

func (r *CatalogRepository) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	query := `SELECT id, name, kind, status, stockable, created_at, updated_at FROM catalog_items WHERE 1=1`
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
		query += fmt.Sprintf(` AND name ILIKE $%d`, n)
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.Stockable, &item.CreatedAt, &item.UpdatedAt); err != nil {
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
		SELECT id, name, kind, status, stockable, created_at, updated_at
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.Stockable, &item.CreatedAt, &item.UpdatedAt); err != nil {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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

func (r *CatalogRepository) UpdateLineLink(
	ctx context.Context,
	lineID string,
	itemID *string,
	status, method string,
	locked bool,
	suggestions []domain.Suggestion,
) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	if suggestions == nil {
		suggestions = []domain.Suggestion{}
	}
	raw, err := json.Marshal(suggestions)
	if err != nil {
		return fmt.Errorf("marshal suggestions: %w", err)
	}
	tag, err := pool.Exec(ctx, `
		UPDATE invoice_lines
		SET item_id = $2, link_status = $3, link_method = NULLIF($4, ''), link_locked = $5,
		    suggestions = $6::jsonb, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, lineID, itemID, status, method, locked, raw)
	if err != nil {
		return fmt.Errorf("update line link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "invoice line not found")
	}
	return nil
}

func (r *CatalogRepository) ListReviewLines(ctx context.Context, statuses []string) ([]ports.ReviewLine, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	if len(statuses) == 0 {
		statuses = []string{domain.LinkStatusUnmatched, domain.LinkStatusSuggested}
	}
	rows, err := pool.Query(ctx, `
		SELECT l.id, l.invoice_header_id, l.line_number, COALESCE(l.item_code, ''), COALESCE(l.description, ''),
			COALESCE(l.item_id, ''), l.link_status, COALESCE(l.link_method, ''), l.link_locked, l.suggestions
		FROM invoice_lines l
		WHERE l.link_status = ANY($1)
		ORDER BY l.updated_at DESC
		LIMIT 200
	`, statuses)
	if err != nil {
		return nil, fmt.Errorf("list review lines: %w", err)
	}
	defer rows.Close()
	out := make([]ports.ReviewLine, 0)
	for rows.Next() {
		var line ports.ReviewLine
		var suggestionsRaw []byte
		if err := rows.Scan(
			&line.LineID, &line.InvoiceHeaderID, &line.LineNumber, &line.ItemCode, &line.Description,
			&line.ItemID, &line.LinkStatus, &line.LinkMethod, &line.LinkLocked, &suggestionsRaw,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(suggestionsRaw, &line.Suggestions)
		out = append(out, line)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) GetLineLinkState(ctx context.Context, lineID string) (*ports.LineLinkState, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	var state ports.LineLinkState
	err = pool.QueryRow(ctx, `
		SELECT l.id, l.invoice_header_id, COALESCE(l.item_id, ''), l.link_status, COALESCE(l.link_method, ''), l.link_locked,
			COALESCE(l.item_code, ''), COALESCE(l.description, ''), COALESCE(h.issuer_party_id, '')
		FROM invoice_lines l
		JOIN invoice_headers h ON h.id = l.invoice_header_id
		WHERE l.id = $1
	`, lineID).Scan(
		&state.LineID, &state.InvoiceHeaderID, &state.ItemID, &state.LinkStatus, &state.LinkMethod, &state.LinkLocked,
		&state.ItemCode, &state.Description, &state.PartyID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get line link state: %w", err)
	}
	return &state, nil
}

func (r *CatalogRepository) SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	// linked when no open review work remains; otherwise pending (user has acted on a failed/pending invoice).
	_, err = pool.Exec(ctx, `
		UPDATE invoice_headers h
		SET linking_status = CASE
			WHEN EXISTS (
				SELECT 1 FROM invoice_lines l
				WHERE l.invoice_header_id = h.id
				  AND l.link_status IN ('unmatched', 'suggested')
			) THEN 'pending'
			ELSE 'linked'
		END,
		updated_at = CURRENT_TIMESTAMP
		WHERE h.id = $1
	`, invoiceHeaderID)
	if err != nil {
		return fmt.Errorf("sync invoice header linking status: %w", err)
	}
	return nil
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
