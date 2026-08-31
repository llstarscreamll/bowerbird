package adapters

import (
	"context"
	"encoding/json"

	catalogApp "github.com/bowerbird/internal/catalog/application"
	catalogDomain "github.com/bowerbird/internal/catalog/domain"
	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	partiesApp "github.com/bowerbird/internal/parties/application"
)

type PartyResolverAdapter struct {
	lookup partiesApp.IssuerPartyLookup
}

func NewPartyResolverAdapter(lookup partiesApp.IssuerPartyLookup) *PartyResolverAdapter {
	return &PartyResolverAdapter{lookup: lookup}
}

func (a *PartyResolverAdapter) ResolveIssuerPartyID(ctx context.Context, taxID, name string) (string, error) {
	if a == nil || a.lookup == nil {
		return "", nil
	}
	return a.lookup.ResolveIssuerPartyID(ctx, taxID, name)
}

var _ ports.IssuerPartyResolver = (*PartyResolverAdapter)(nil)

type CatalogResolverAdapter struct {
	app *catalogApp.Application
}

func NewCatalogResolverAdapter(app *catalogApp.Application) *CatalogResolverAdapter {
	return &CatalogResolverAdapter{app: app}
}

func (a *CatalogResolverAdapter) ResolveLine(ctx context.Context, input ports.CatalogLineResolveInput) (*ports.CatalogLineResolveResult, error) {
	if a == nil || a.app == nil || a.app.Commands.ResolveInvoiceLine == nil {
		return &ports.CatalogLineResolveResult{Status: domain.LinkStatusUnmatched}, nil
	}
	result, err := a.app.Commands.ResolveInvoiceLine.Execute(ctx, catalogDomain.LineResolutionInput{
		LineID:         input.LineID,
		PartyID:        input.PartyID,
		ItemCode:       input.ItemCode,
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
	suggestionsJSON := []byte("[]")
	if result.Suggestions != nil {
		suggestionsJSON, err = json.Marshal(result.Suggestions)
		if err != nil {
			return nil, err
		}
	}
	return &ports.CatalogLineResolveResult{
		ItemID:      result.ItemID,
		Status:      result.Status,
		Method:      result.Method,
		Suggestions: suggestionsJSON,
	}, nil
}

var _ ports.CatalogLineResolver = (*CatalogResolverAdapter)(nil)

type CatalogNamesAdapter struct {
	app *catalogApp.Application
}

func NewCatalogNamesAdapter(app *catalogApp.Application) *CatalogNamesAdapter {
	return &CatalogNamesAdapter{app: app}
}

func (a *CatalogNamesAdapter) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	if a == nil || a.app == nil || a.app.Queries.GetItemNames == nil {
		return map[string]string{}, nil
	}
	return a.app.Queries.GetItemNames.Execute(ctx, ids)
}

var _ ports.CatalogService = (*CatalogNamesAdapter)(nil)
