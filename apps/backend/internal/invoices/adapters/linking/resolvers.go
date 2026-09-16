package adapters

import (
	"context"
	"encoding/json"

	catalogapi "github.com/bowerbird/internal/catalog/api"
	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	partiesapi "github.com/bowerbird/internal/parties/api"
)

type PartyResolverAdapter struct {
	lookup partiesapi.IssuerPartyLookup
}

func NewPartyResolverAdapter(lookup partiesapi.IssuerPartyLookup) *PartyResolverAdapter {
	if lookup == nil {
		panic("issuer party lookup is required")
	}
	return &PartyResolverAdapter{lookup: lookup}
}

func (a *PartyResolverAdapter) ResolveIssuer(ctx context.Context, profile ports.IssuerProfile) (string, error) {
	addrs := make([]partiesapi.IssuerAddress, 0, len(profile.Addresses))
	for _, addr := range profile.Addresses {
		addrs = append(addrs, partiesapi.IssuerAddress{
			Line: addr.Line, City: addr.City, Department: addr.Department,
			PostalZone: addr.PostalZone, CountryCode: addr.CountryCode, Kind: addr.Kind,
		})
	}
	return a.lookup.ResolveIssuer(ctx, partiesapi.IssuerProfile{
		TaxID: profile.TaxID, Name: profile.Name, SchemeID: profile.SchemeID,
		TaxpayerKind: profile.TaxpayerKind, TaxLevelCode: profile.TaxLevelCode,
		Emails: profile.Emails, Phones: profile.Phones, Addresses: addrs,
	})
}

var _ ports.IssuerPartyResolver = (*PartyResolverAdapter)(nil)

type CatalogACL struct {
	catalog catalogapi.InvoiceSupport
}

func NewCatalogACL(catalog catalogapi.InvoiceSupport) *CatalogACL {
	if catalog == nil {
		panic("catalog invoice support is required")
	}
	return &CatalogACL{catalog: catalog}
}

func (a *CatalogACL) ResolveLine(ctx context.Context, input ports.CatalogLineResolveInput) (*ports.CatalogLineResolveResult, error) {
	result, err := a.catalog.ResolveLine(ctx, catalogapi.LineResolveInput{
		LineID:         input.LineID,
		PartyID:        input.PartyID,
		BuyerCode:      input.BuyerCode,
		SellerSKU:      input.SellerSKU,
		GTIN:           input.GTIN,
		Description:    input.Description,
		ExistingItemID: input.ExistingItemID,
		ExistingLocked: input.ExistingLocked,
		ExistingStatus: input.ExistingStatus,
		ExistingMethod: input.ExistingMethod,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &ports.CatalogLineResolveResult{Status: domain.LinkStatusUnmatched}, nil
	}
	suggestionsJSON, err := json.Marshal(result.Suggestions)
	if err != nil {
		return nil, err
	}
	if len(result.Suggestions) == 0 {
		suggestionsJSON = []byte("[]")
	}
	return &ports.CatalogLineResolveResult{
		ItemID:      result.ItemID,
		Status:      result.Status,
		Method:      result.Method,
		Suggestions: suggestionsJSON,
	}, nil
}

func (a *CatalogACL) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	return a.catalog.GetItemNames(ctx, ids)
}

func (a *CatalogACL) GetItemDisplays(ctx context.Context, ids []string) (map[string]ports.ItemDisplay, error) {
	raw, err := a.catalog.GetItemDisplays(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ports.ItemDisplay, len(raw))
	for id, d := range raw {
		out[id] = ports.ItemDisplay{Name: d.Name, InternalCode: d.InternalCode}
	}
	return out, nil
}

func (a *CatalogACL) ValidateItemExists(ctx context.Context, itemID string) error {
	return a.catalog.ValidateItemExists(ctx, itemID)
}

func (a *CatalogACL) MintProvisionalFromEvidence(ctx context.Context, input ports.MintProvisionalInput) (string, error) {
	return a.catalog.MintProvisionalFromEvidence(ctx, catalogapi.MintFromEvidenceInput{
		PartyID:     input.PartyID,
		SellerSKU:   input.SellerSKU,
		GTIN:        input.GTIN,
		Description: input.Description,
	})
}

func (a *CatalogACL) RememberDecision(ctx context.Context, input ports.RememberDecisionInput) error {
	return a.catalog.RememberDecision(ctx, catalogapi.RememberDecisionInput{
		PartyID:     input.PartyID,
		SellerSKU:   input.SellerSKU,
		GTIN:        input.GTIN,
		Description: input.Description,
		Action:      input.Action,
		ItemID:      input.ItemID,
	})
}

var _ ports.CatalogLineResolver = (*CatalogACL)(nil)
var _ ports.CatalogService = (*CatalogACL)(nil)
var _ ports.CatalogMatchingPort = (*CatalogACL)(nil)
