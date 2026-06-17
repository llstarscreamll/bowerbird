package ports

import (
	"context"

	"github.com/bowerbird/internal/invoices/domain"
)

type InvoiceWriteRepository interface {
	PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error
}

type InvoiceRepository interface {
	InvoiceWriteRepository
	ExistsBySource(ctx context.Context, sourceName string, sourceID string) (bool, error)
	ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error)
}
