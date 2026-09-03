package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	registry *database.Registry
}

func NewRepository(registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{registry: registry}
}

func (r *PostgresRepository) ExistsBySource(ctx context.Context, sourceName string, sourceID string) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("get tenant db pool: %w", err)
	}

	var exists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoice_headers WHERE source_name = $1 AND source_id = $2)`, sourceName, sourceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check invoice by source: %w", err)
	}

	return exists, nil
}

func (r *PostgresRepository) ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("get tenant db pool: %w", err)
	}

	var exists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoice_headers WHERE cufe = $1)`, cufe).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check invoice by cufe: %w", err)
	}

	return exists, nil
}

func (r *PostgresRepository) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin invoice transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	headRaw := header.RawData
	if len(headRaw) == 0 {
		headRaw = []byte("{}")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_headers (
			id, source_name, source_id, cufe, invoice_number, issuer_name,
			issuer_tax_id, receiver_name, receiver_tax_id, currency_code, issue_date,
			due_date, payment_code, subtotal, tax_total, allowance_total, grand_total,
			document_ref_s3_key, extraction_source, raw_data, created_at, updated_at,
			issuer_party_id, linking_status
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22,
			NULLIF($23, ''), COALESCE(NULLIF($24, ''), 'pending')
		)
	`,
		header.ID,
		header.SourceName,
		header.SourceID,
		header.CUFE,
		header.InvoiceNumber,
		header.IssuerName,
		header.IssuerTaxID,
		header.ReceiverName,
		header.ReceiverTaxID,
		header.CurrencyCode,
		header.IssueDate,
		header.DueDate,
		header.PaymentCode,
		header.Subtotal,
		header.TaxTotal,
		header.AllowanceTotal,
		header.GrandTotal,
		header.DocumentRefS3Key,
		header.ExtractionSource,
		headRaw,
		header.CreatedAt,
		header.UpdatedAt,
		header.IssuerPartyID,
		header.LinkingStatus,
	); err != nil {
		return fmt.Errorf("insert invoice header: %w", err)
	}

	for _, line := range lines {
		lineRaw := line.RawData
		if len(lineRaw) == 0 {
			lineRaw = []byte("{}")
		}
		suggestions := line.Suggestions
		if len(suggestions) == 0 {
			suggestions = []byte("[]")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_lines (
				id, invoice_header_id, line_number, item_code, description,
				quantity, unit_price, line_tax_total, line_total,
				raw_data, created_at, updated_at,
				item_id, link_status, link_method, link_locked, suggestions
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9,
				$10, $11, $12,
				NULLIF($13, ''), COALESCE(NULLIF($14, ''), 'unmatched'), NULLIF($15, ''), $16, $17::jsonb
			)
		`,
			line.ID,
			line.InvoiceHeaderID,
			line.LineNumber,
			line.ItemCode,
			line.Description,
			line.Quantity,
			line.UnitPrice,
			line.LineTaxTotal,
			line.LineTotal,
			lineRaw,
			line.CreatedAt,
			line.UpdatedAt,
			line.ItemID,
			line.LinkStatus,
			line.LinkMethod,
			line.LinkLocked,
			suggestions,
		); err != nil {
			return fmt.Errorf("insert invoice line: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit invoice transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ApplyCatalogLinking(
	ctx context.Context,
	headerID string,
	issuerPartyID *string,
	linkingStatus string,
	lines []ports.LineLinkUpdate,
) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin linking transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE invoice_headers
		SET issuer_party_id = $2, linking_status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, headerID, issuerPartyID, linkingStatus); err != nil {
		return fmt.Errorf("update invoice header linking: %w", err)
	}

	for _, line := range lines {
		suggestions := line.Suggestions
		if len(suggestions) == 0 {
			suggestions = []byte("[]")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE invoice_lines
			SET item_id = $2, link_status = $3, link_method = NULLIF($4, ''),
			    link_locked = $5, suggestions = $6::jsonb, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, line.LineID, line.ItemID, line.LinkStatus, line.LinkMethod, line.LinkLocked, suggestions); err != nil {
			return fmt.Errorf("update invoice line linking: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit linking transaction: %w", err)
	}
	return nil
}

var _ ports.InvoiceRepository = (*PostgresRepository)(nil)

func (r *PostgresRepository) GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	var header domain.InvoiceHeaderRecord
	err = pool.QueryRow(ctx, `
		SELECT id, source_name, source_id, cufe, invoice_number, issuer_name,
			issuer_tax_id, receiver_name, receiver_tax_id, currency_code, issue_date,
			due_date, payment_code, subtotal, tax_total, allowance_total, grand_total,
			document_ref_s3_key, extraction_source, raw_data, created_at, updated_at,
			COALESCE(issuer_party_id, ''), COALESCE(linking_status, 'pending')
		FROM invoice_headers
		WHERE id = $1
	`, id).Scan(
		&header.ID, &header.SourceName, &header.SourceID, &header.CUFE, &header.InvoiceNumber,
		&header.IssuerName, &header.IssuerTaxID, &header.ReceiverName, &header.ReceiverTaxID,
		&header.CurrencyCode, &header.IssueDate, &header.DueDate, &header.PaymentCode,
		&header.Subtotal, &header.TaxTotal, &header.AllowanceTotal, &header.GrandTotal, &header.DocumentRefS3Key,
		&header.ExtractionSource, &header.RawData, &header.CreatedAt, &header.UpdatedAt,
		&header.IssuerPartyID, &header.LinkingStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("invoice not found")
		}
		return nil, nil, fmt.Errorf("get invoice header: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT id, invoice_header_id, line_number, item_code, description,
			quantity, unit_price, line_tax_total, line_total, raw_data, created_at, updated_at,
			COALESCE(item_id, ''), COALESCE(link_status, 'unmatched'), COALESCE(link_method, ''),
			COALESCE(link_locked, false), COALESCE(suggestions, '[]'::jsonb)
		FROM invoice_lines
		WHERE invoice_header_id = $1
		ORDER BY line_number
	`, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get invoice lines: %w", err)
	}
	defer rows.Close()

	var lines []domain.InvoiceLineRecord
	for rows.Next() {
		var line domain.InvoiceLineRecord
		if err := rows.Scan(
			&line.ID, &line.InvoiceHeaderID, &line.LineNumber, &line.ItemCode,
			&line.Description, &line.Quantity, &line.UnitPrice, &line.LineTaxTotal,
			&line.LineTotal, &line.RawData, &line.CreatedAt, &line.UpdatedAt,
			&line.ItemID, &line.LinkStatus, &line.LinkMethod, &line.LinkLocked, &line.Suggestions,
		); err != nil {
			return nil, nil, fmt.Errorf("scan invoice line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate invoice lines: %w", err)
	}

	return &header, lines, nil
}

func (r *PostgresRepository) ListInvoices(ctx context.Context, limit int, cursor string) ([]domain.InvoiceHeaderRecord, bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("get tenant db pool: %w", err)
	}

	query := `
		SELECT id, source_name, source_id, cufe, invoice_number, issuer_name,
			issuer_tax_id, receiver_name, receiver_tax_id, currency_code, issue_date,
			due_date, payment_code, subtotal, tax_total, allowance_total, grand_total,
			document_ref_s3_key, extraction_source, raw_data, created_at, updated_at,
			COALESCE(issuer_party_id, ''), COALESCE(linking_status, 'pending')
		FROM invoice_headers
	`
	args := []any{}

	if cursor != "" {
		query += " WHERE id < $1"
		args = append(args, cursor)
	}

	// Request one extra item to determine if there are more pages
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit+1)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var headers []domain.InvoiceHeaderRecord
	for rows.Next() {
		var header domain.InvoiceHeaderRecord
		if err := rows.Scan(
			&header.ID, &header.SourceName, &header.SourceID, &header.CUFE, &header.InvoiceNumber,
			&header.IssuerName, &header.IssuerTaxID, &header.ReceiverName, &header.ReceiverTaxID,
			&header.CurrencyCode, &header.IssueDate, &header.DueDate, &header.PaymentCode,
			&header.Subtotal, &header.TaxTotal, &header.AllowanceTotal, &header.GrandTotal, &header.DocumentRefS3Key,
			&header.ExtractionSource, &header.RawData, &header.CreatedAt, &header.UpdatedAt,
			&header.IssuerPartyID, &header.LinkingStatus,
		); err != nil {
			return nil, false, fmt.Errorf("scan invoice header: %w", err)
		}
		headers = append(headers, header)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate invoices: %w", err)
	}

	hasMore := len(headers) > limit
	if hasMore {
		headers = headers[:limit]
	}

	return headers, hasMore, nil
}
