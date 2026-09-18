package api

import "context"

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

// IssuerPartyLookup is the parties Open Host Service for invoice issuer resolution.
type IssuerPartyLookup interface {
	ResolveIssuer(ctx context.Context, profile IssuerProfile) (partyID string, err error)
}
