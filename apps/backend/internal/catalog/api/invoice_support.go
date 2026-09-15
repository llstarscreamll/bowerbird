package api

import "context"

// InvoiceSupport is the catalog Open Host Service for invoicing.
type InvoiceSupport interface {
	ResolveLine(ctx context.Context, input LineResolveInput) (*LineResolveResult, error)
	ValidateItemExists(ctx context.Context, itemID string) error
	MintProvisionalFromEvidence(ctx context.Context, input MintFromEvidenceInput) (itemID string, err error)
	RememberDecision(ctx context.Context, input RememberDecisionInput) error
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
	GetItemDisplays(ctx context.Context, ids []string) (map[string]ItemDisplay, error)
}

type LineResolveInput struct {
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
	SellerSKU   string
	GTIN        string
	Description string
}

type RememberDecisionInput struct {
	PartyID     string
	SellerSKU   string
	GTIN        string
	Description string
	Action      string
	ItemID      string
}

type ItemDisplay struct {
	Name         string
	InternalCode string
}
