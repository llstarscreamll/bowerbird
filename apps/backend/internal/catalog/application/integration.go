package application

import (
	"context"

	"github.com/bowerbird/internal/catalog/api"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/queries"
	"github.com/bowerbird/internal/catalog/domain"
)

type invoiceSupport struct {
	resolve  *commands.ResolveInvoiceLineCommand
	validate *commands.ValidateCatalogItemCommand
	mint     *commands.MintProvisionalFromEvidenceCommand
	alias    *commands.EnsureSupplierAliasCommand
	memory   *commands.RecordMatchMemoryCommand
	names    *queries.GetItemNamesQuery
	displays *queries.GetItemDisplaysQuery
}

func NewInvoiceSupport(app *Application) api.InvoiceSupport {
	if app == nil {
		panic("catalog application is required")
	}
	if app.Commands.ResolveInvoiceLine == nil {
		panic("resolve invoice line command is required")
	}
	if app.Commands.ValidateCatalogItem == nil {
		panic("validate catalog item command is required")
	}
	if app.Commands.MintProvisionalFromEvidence == nil {
		panic("mint provisional command is required")
	}
	if app.Commands.EnsureSupplierAlias == nil {
		panic("ensure supplier alias command is required")
	}
	if app.Commands.RecordMatchMemory == nil {
		panic("record match memory command is required")
	}
	if app.Queries.GetItemNames == nil {
		panic("get item names query is required")
	}
	if app.Queries.GetItemDisplays == nil {
		panic("get item displays query is required")
	}
	return &invoiceSupport{
		resolve:  app.Commands.ResolveInvoiceLine,
		validate: app.Commands.ValidateCatalogItem,
		mint:     app.Commands.MintProvisionalFromEvidence,
		alias:    app.Commands.EnsureSupplierAlias,
		memory:   app.Commands.RecordMatchMemory,
		names:    app.Queries.GetItemNames,
		displays: app.Queries.GetItemDisplays,
	}
}

func (s *invoiceSupport) ResolveLine(ctx context.Context, input api.LineResolveInput) (*api.LineResolveResult, error) {
	result, err := s.resolve.Execute(ctx, domain.LineResolutionInput{
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
		return &api.LineResolveResult{}, nil
	}
	suggestions := make([]api.LineSuggestion, 0, len(result.Suggestions))
	for _, suggestion := range result.Suggestions {
		suggestions = append(suggestions, api.LineSuggestion{
			ItemID: suggestion.ItemID,
			Score:  suggestion.Score,
			Reason: suggestion.Reason,
		})
	}
	return &api.LineResolveResult{
		ItemID:      result.ItemID,
		Status:      result.Status,
		Method:      result.Method,
		Suggestions: suggestions,
	}, nil
}

func (s *invoiceSupport) ValidateItemExists(ctx context.Context, itemID string) error {
	return s.validate.Execute(ctx, itemID)
}

func (s *invoiceSupport) MintProvisionalFromEvidence(ctx context.Context, input api.MintFromEvidenceInput) (string, error) {
	return s.mint.Execute(ctx, commands.MintProvisionalFromEvidenceInput{
		PartyID:     input.PartyID,
		ItemCode:    input.ItemCode,
		Description: input.Description,
	})
}

func (s *invoiceSupport) EnsureSupplierAlias(ctx context.Context, partyID, itemCode, itemID string) error {
	return s.alias.Execute(ctx, partyID, itemCode, itemID)
}

func (s *invoiceSupport) RecordMatchMemory(ctx context.Context, input api.MatchMemoryInput) error {
	return s.memory.Execute(ctx, commands.RecordMatchMemoryInput{
		PartyID:     input.PartyID,
		ItemCode:    input.ItemCode,
		Description: input.Description,
		Action:      input.Action,
		ItemID:      input.ItemID,
	})
}

func (s *invoiceSupport) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	return s.names.Execute(ctx, ids)
}

func (s *invoiceSupport) GetItemDisplays(ctx context.Context, ids []string) (map[string]api.ItemDisplay, error) {
	raw, err := s.displays.Execute(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]api.ItemDisplay, len(raw))
	for id, display := range raw {
		out[id] = api.ItemDisplay{Name: display.Name, InternalSKU: display.InternalSKU}
	}
	return out, nil
}

var _ api.InvoiceSupport = (*invoiceSupport)(nil)
