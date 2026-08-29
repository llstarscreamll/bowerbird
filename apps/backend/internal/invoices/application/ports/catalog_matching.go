package ports

import "context"

type MintProvisionalInput struct {
	PartyID     string
	ItemCode    string
	Description string
}

type MatchMemoryInput struct {
	PartyID     string
	ItemCode    string
	Description string
	Action      string // link | never_match
	ItemID      *string
}

// CatalogMatchingPort is the anti-corruption boundary for catalog identity and match memory effects.
type CatalogMatchingPort interface {
	ValidateItemExists(ctx context.Context, itemID string) error
	MintProvisionalFromEvidence(ctx context.Context, input MintProvisionalInput) (itemID string, err error)
	EnsureSupplierAlias(ctx context.Context, partyID, itemCode, itemID string) error
	RecordMatchMemory(ctx context.Context, input MatchMemoryInput) error
}
