package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/jackc/pgx/v5"
)

var _ ports.InvoiceLineLinkRepository = (*PostgresRepository)(nil)

func (r *PostgresRepository) SaveLineLink(ctx context.Context, lineID string, link domain.LineLink) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	itemID, status, method, locked, suggestions := link.PersistFields()
	tag, err := pool.Exec(ctx, `
		UPDATE invoice_lines
		SET item_id = $2, link_status = $3, link_method = NULLIF($4, ''), link_locked = $5,
		    suggestions = $6::jsonb, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, lineID, itemID, status, method, locked, suggestions)
	if err != nil {
		return fmt.Errorf("update line link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "invoice line not found")
	}
	return nil
}

func (r *PostgresRepository) ListReviewLines(ctx context.Context, statuses []string) ([]ports.ReviewLine, error) {
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
		var suggestions []suggestionPayload
		_ = json.Unmarshal(suggestionsRaw, &suggestions)
		if suggestions == nil {
			suggestions = []suggestionPayload{}
		}
		enriched := make([]ports.EnrichedSuggestion, 0, len(suggestions))
		for _, s := range suggestions {
			enriched = append(enriched, ports.EnrichedSuggestion{
				ItemID: s.ItemID,
				Score:  s.Score,
				Reason: s.Reason,
			})
		}
		line.Suggestions = enriched
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PostgresRepository) GetLineForDecision(ctx context.Context, lineID string) (*domain.LineForDecision, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	var state domain.LineForDecision
	var itemID, status, method string
	var locked bool
	err = pool.QueryRow(ctx, `
		SELECT l.id, l.invoice_header_id, COALESCE(l.item_id, ''), l.link_status, COALESCE(l.link_method, ''), l.link_locked,
			COALESCE(l.item_code, ''), COALESCE(l.description, ''), COALESCE(h.issuer_party_id, '')
		FROM invoice_lines l
		JOIN invoice_headers h ON h.id = l.invoice_header_id
		WHERE l.id = $1
	`, lineID).Scan(
		&state.LineID, &state.InvoiceHeaderID, &itemID, &status, &method, &locked,
		&state.ItemCode, &state.Description, &state.PartyID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get line for decision: %w", err)
	}
	state.Link = domain.LineLinkFromRecord(itemID, status, method, locked, nil)
	return &state, nil
}

func (r *PostgresRepository) SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT link_status FROM invoice_lines WHERE invoice_header_id = $1
	`, invoiceHeaderID)
	if err != nil {
		return fmt.Errorf("list line link statuses: %w", err)
	}
	defer rows.Close()

	statuses := make([]string, 0)
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return err
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	linkingStatus := domain.RecalculateLinkingStatus(statuses)
	_, err = pool.Exec(ctx, `
		UPDATE invoice_headers
		SET linking_status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, invoiceHeaderID, linkingStatus)
	if err != nil {
		return fmt.Errorf("sync invoice header linking status: %w", err)
	}
	return nil
}

type suggestionPayload struct {
	ItemID string  `json:"item_id"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}
