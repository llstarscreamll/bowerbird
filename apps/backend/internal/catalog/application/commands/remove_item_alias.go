package commands

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type RemoveItemAliasCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
}

func NewRemoveItemAliasCommand(items ports.ItemRepository, aliases ports.AliasRepository) *RemoveItemAliasCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	return &RemoveItemAliasCommand{items: items, aliases: aliases}
}

func (cmd *RemoveItemAliasCommand) Execute(ctx context.Context, itemID, aliasID string) error {
	item, err := cmd.items.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return cmd.aliases.DeleteAlias(ctx, itemID, aliasID)
}
