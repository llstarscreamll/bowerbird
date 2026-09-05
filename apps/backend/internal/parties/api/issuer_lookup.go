package api

import "context"

// IssuerPartyLookup is the parties Open Host Service for invoice issuer resolution.
type IssuerPartyLookup interface {
	ResolveIssuerPartyID(ctx context.Context, taxID, name string) (partyID string, err error)
}
