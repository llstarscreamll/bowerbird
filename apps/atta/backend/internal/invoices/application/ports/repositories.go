package ports

import (
	"context"

	"github.com/atta/internal/invoices/domain"
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
	ResolveIssuer(ctx context.Context, profile IssuerProfile) (partyID string, err error)
}

type IssuerAddress struct {
	Line        string
	City        string
	Department  string
	PostalZone  string
	CountryCode string
	Kind        string
}

type IssuerProfile struct {
	TaxID        string
	Name         string
	SchemeID     string
	TaxpayerKind string
	TaxLevelCode string
	Emails       []string
	Phones       []string
	Addresses    []IssuerAddress
}

func IssuerProfileFromParty(party domain.Party) IssuerProfile {
	addrs := make([]IssuerAddress, 0, len(party.Addresses))
	for _, a := range party.Addresses {
		addrs = append(addrs, IssuerAddress{
			Line: a.Line, City: a.City, Department: a.Department,
			PostalZone: a.PostalZone, CountryCode: a.CountryCode, Kind: a.Kind,
		})
	}
	return IssuerProfile{
		TaxID: party.TaxID, Name: party.Name, SchemeID: party.SchemeID,
		TaxpayerKind: party.TaxpayerKind, TaxLevelCode: party.TaxLevelCode,
		Emails: party.Emails, Phones: party.Phones, Addresses: addrs,
	}
}

type CatalogLineResolveInput struct {
	LineID         string
	PartyID        string
	BuyerCode      string
	SellerSKU      string
	GTIN           string
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

type ReceiverDirectory interface {
	HasAny(ctx context.Context) (bool, error)
	ReceiverMatches(ctx context.Context, taxID string) (bool, error)
}

type ItemIDPair struct {
	Left  string
	Right string
}

type CatalogItemLinkRepository interface {
	RelinkCatalogItems(ctx context.Context, fromIDs []string, toID string) error
	HardConflictItemPairs(ctx context.Context) ([]ItemIDPair, error)
	CountLinesByItemIDs(ctx context.Context, itemIDs []string) (map[string]int, error)
}
