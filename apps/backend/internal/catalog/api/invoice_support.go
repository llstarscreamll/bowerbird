package api

import "context"

// InvoiceSupport is the catalog Open Host Service for invoicing.
type InvoiceSupport interface {
	ResolveLine(ctx context.Context, input LineResolveInput) (*LineResolveResult, error)
	ValidateItemExists(ctx context.Context, itemID string) error
	MintProvisionalFromEvidence(ctx context.Context, input MintFromEvidenceInput) (itemID string, err error)
	EnsureSupplierAlias(ctx context.Context, partyID, itemCode, itemID string) error
	RecordMatchMemory(ctx context.Context, input MatchMemoryInput) error
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
	GetItemDisplays(ctx context.Context, ids []string) (map[string]ItemDisplay, error)
}

type LineResolveInput struct {
	LineID         string
	PartyID        string
	ItemCode       string
	Description    string
	ExistingItemID string
	ExistingLocked bool
	ExistingStatus string
	ExistingMethod string
}

type LineSuggestion struct {
	ItemID string
	Score  float64
	Reason string
}

type LineResolveResult struct {
	ItemID      string
	Status      string
	Method      string
	Suggestions []LineSuggestion
}

type MintFromEvidenceInput struct {
	PartyID     string
	ItemCode    string
	Description string
}

type MatchMemoryInput struct {
	PartyID     string
	ItemCode    string
	Description string
	Action      string
	ItemID      *string
}

type ItemDisplay struct {
	Name        string
	InternalSKU string
}
