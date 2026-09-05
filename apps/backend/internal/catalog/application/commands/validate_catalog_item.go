package commands

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/catalog/application/ports"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type ValidateCatalogItemCommand struct {
	items ports.ItemRepository
}

func NewValidateCatalogItemCommand(items ports.ItemRepository) *ValidateCatalogItemCommand {
	return &ValidateCatalogItemCommand{items: items}
}

func (cmd *ValidateCatalogItemCommand) Execute(ctx context.Context, itemID string) error {
	if cmd.items == nil {
		return fmt.Errorf("item repository is required")
	}
	item, err := cmd.items.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return nil
}
