package adapters

import (
	"context"
	"encoding/json"

	catalogApp "github.com/bowerbird/internal/catalog/application"
	catalogDomain "github.com/bowerbird/internal/catalog/domain"
	"github.com/bowerbird/internal/invoices/application/ports"
	partiesApp "github.com/bowerbird/internal/parties/application"
)

type PartyResolverAdapter struct {
	app *partiesApp.Application
}

func NewPartyResolverAdapter(app *partiesApp.Application) *PartyResolverAdapter {
	return &PartyResolverAdapter{app: app}
}

func (a *PartyResolverAdapter) ResolveIssuerPartyID(ctx context.Context, taxID, name string) (string, error) {
	if a == nil || a.app == nil || a.app.Commands.ResolveOrCreateFromIssuer == nil {
		return "", nil
	}
	party, err := a.app.Commands.ResolveOrCreateFromIssuer.Execute(ctx, taxID, name)
	if err != nil {
		return "", err
	}
	if party == nil {
		return "", nil
	}
	return party.ID, nil
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
		return &ports.CatalogLineResolveResult{Status: catalogDomain.LinkStatusUnmatched}, nil
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
	suggestions := result.Suggestions
	if suggestions == nil {
		suggestions = []catalogDomain.Suggestion{}
	}
	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		return nil, err
	}
	return &ports.CatalogLineResolveResult{
		ItemID:      result.ItemID,
		Status:      result.Status,
		Method:      result.Method,
		Suggestions: suggestionsJSON,
	}, nil
}

var _ ports.CatalogLineResolver = (*CatalogResolverAdapter)(nil)
