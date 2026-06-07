package postgres

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/database"
)

type PostgresRepository struct {
	registry *database.Registry
}

func NewRepository(registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{registry: registry}
}

func (r *PostgresRepository) ExistsInvoiceBySourceMessageID(ctx context.Context, sourceMessageID string) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("get tenant db pool: %w", err)
	}

	var exists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM invoice_headers WHERE source_message_id = $1)`, sourceMessageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check invoice by source message id: %w", err)
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
			id, source_message_id, cufe, invoice_number, issuer_name, issuer_tax_id,
			receiver_name, receiver_tax_id, currency_code, issue_date, due_date,
			payment_code, subtotal, tax_total, grand_total, document_ref_s3_key,
			extraction_source, raw_data, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20
		)
	`,
		header.ID,
		header.SourceMessageID,
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
		header.GrandTotal,
		header.DocumentRefS3Key,
		header.ExtractionSource,
		headRaw,
		header.CreatedAt,
		header.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert invoice header: %w", err)
	}

	for _, line := range lines {
		lineRaw := line.RawData
		if len(lineRaw) == 0 {
			lineRaw = []byte("{}")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_lines (
				id, invoice_header_id, line_number, item_code, description,
				quantity, unit_price, line_tax_total, line_total,
				raw_data, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9,
				$10, $11, $12
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
		); err != nil {
			return fmt.Errorf("insert invoice line: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit invoice transaction: %w", err)
	}

	return nil
}

var _ ports.InvoiceRepository = (*PostgresRepository)(nil)
