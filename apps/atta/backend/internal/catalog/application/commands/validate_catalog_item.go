package commands

import (
	"context"

	"github.com/atta/internal/catalog/application/ports"
	appErrors "github.com/atta/internal/platform/errors"
)

type ValidateCatalogItemCommand struct {
	items ports.ItemRepository
}

func NewValidateCatalogItemCommand(items ports.ItemRepository) *ValidateCatalogItemCommand {
	if items == nil {
		panic("item repository is required")
	}
	return &ValidateCatalogItemCommand{items: items}
}

func (cmd *ValidateCatalogItemCommand) Execute(ctx context.Context, itemID string) error {
	item, err := cmd.items.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	if item.IsMerged() {
		return appErrors.New(appErrors.CodeGone, "catalog item was merged").WithMeta("merged_into_id", item.MergedIntoID)
	}
	return nil
}
