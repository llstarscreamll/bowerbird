package ports

import (
	"context"

	"github.com/bowerbird/internal/invoices/domain"
)

type InvoiceWriteRepository interface {
	PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error
	ApplyCatalogLinking(ctx context.Context, headerID string, issuerPartyID *string, linkingStatus string, lines []LineLinkUpdate) error
}

type LineLinkUpdate struct {
	LineID      string
	ItemID      *string
	LinkStatus  string
	LinkMethod  string
	LinkLocked  bool
	Suggestions []byte
}

func NewLineLinkUpdate(lineID string, link domain.LineLink) LineLinkUpdate {
	itemID, status, method, locked, suggestions := link.PersistFields()
	return LineLinkUpdate{
		LineID:      lineID,
		ItemID:      itemID,
		LinkStatus:  status,
		LinkMethod:  method,
		LinkLocked:  locked,
		Suggestions: suggestions,
	}
}

type InvoiceQueryRepository interface {
	GetInvoiceByID(ctx context.Context, id string) (*domain.InvoiceHeaderRecord, []domain.InvoiceLineRecord, error)
	ListInvoices(ctx context.Context, limit int, cursor string) ([]domain.InvoiceHeaderRecord, bool, error)
}

type InvoiceRepository interface {
	InvoiceWriteRepository
	InvoiceQueryRepository
	ExistsBySource(ctx context.Context, sourceName string, sourceID string) (bool, error)
	ExistsInvoiceByCUFE(ctx context.Context, cufe string) (bool, error)
}

// IssuerPartyResolver resolves or creates a party from invoice issuer fields.
// Returns empty partyID when tax id is missing.
type IssuerPartyResolver interface {
	ResolveIssuerPartyID(ctx context.Context, taxID, name string) (partyID string, err error)
}

type CatalogLineResolveInput struct {
	LineID         string
	PartyID        string
	ItemCode       string
	Description    string
	ExistingItemID string
	ExistingLocked bool
	ExistingStatus string
	ExistingMethod string
}

type CatalogLineResolveResult struct {
	ItemID      string
	Status      string
	Method      string
	Suggestions []byte
}

type CatalogLineResolver interface {
	ResolveLine(ctx context.Context, input CatalogLineResolveInput) (*CatalogLineResolveResult, error)
}
