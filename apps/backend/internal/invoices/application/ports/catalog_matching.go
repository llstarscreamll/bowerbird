package ports

import "context"

type MintProvisionalInput struct {
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

// CatalogMatchingPort is the anti-corruption boundary for catalog identity and match memory.
type CatalogMatchingPort interface {
	ValidateItemExists(ctx context.Context, itemID string) error
	MintProvisionalFromEvidence(ctx context.Context, input MintProvisionalInput) (itemID string, err error)
	RememberDecision(ctx context.Context, input RememberDecisionInput) error
}
