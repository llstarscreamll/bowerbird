package adapters

import (
	"context"
	"fmt"
	"log/slog"

	catalogApp "github.com/bowerbird/internal/catalog/application"
	catalogCommands "github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/invoices/application/ports"
)

type CatalogMatchingAdapter struct {
	app    *catalogApp.Application
	logger *slog.Logger
}

func NewCatalogMatchingAdapter(app *catalogApp.Application) *CatalogMatchingAdapter {
	return &CatalogMatchingAdapter{app: app, logger: slog.Default()}
}

var _ ports.CatalogMatchingPort = (*CatalogMatchingAdapter)(nil)

func (a *CatalogMatchingAdapter) ValidateItemExists(ctx context.Context, itemID string) error {
	if a.app == nil || a.app.Commands.ValidateCatalogItem == nil {
		return fmt.Errorf("catalog validate item command is required")
	}
	return a.app.Commands.ValidateCatalogItem.Execute(ctx, itemID)
}

func (a *CatalogMatchingAdapter) MintProvisionalFromEvidence(ctx context.Context, input ports.MintProvisionalInput) (string, error) {
	if a.app == nil || a.app.Commands.MintProvisionalFromEvidence == nil {
		return "", fmt.Errorf("catalog mint provisional command is required")
	}
	itemID, err := a.app.Commands.MintProvisionalFromEvidence.Execute(ctx, catalogCommands.MintProvisionalFromEvidenceInput{
		PartyID:     input.PartyID,
		ItemCode:    input.ItemCode,
		Description: input.Description,
	})
	if err != nil {
		return "", err
	}
	a.logger.Info("catalog.mint_provisional", "item_id", itemID)
	return itemID, nil
}

func (a *CatalogMatchingAdapter) EnsureSupplierAlias(ctx context.Context, partyID, itemCode, itemID string) error {
	if a.app == nil || a.app.Commands.EnsureSupplierAlias == nil {
		return fmt.Errorf("catalog ensure supplier alias command is required")
	}
	return a.app.Commands.EnsureSupplierAlias.Execute(ctx, partyID, itemCode, itemID)
}

func (a *CatalogMatchingAdapter) RecordMatchMemory(ctx context.Context, input ports.MatchMemoryInput) error {
	if a.app == nil || a.app.Commands.RecordMatchMemory == nil {
		return fmt.Errorf("catalog record match memory command is required")
	}
	if err := a.app.Commands.RecordMatchMemory.Execute(ctx, catalogCommands.RecordMatchMemoryInput{
		PartyID:     input.PartyID,
		ItemCode:    input.ItemCode,
		Description: input.Description,
		Action:      input.Action,
		ItemID:      input.ItemID,
	}); err != nil {
		return err
	}
	a.logger.Info("catalog.record_match_memory", "action", input.Action)
	return nil
}
