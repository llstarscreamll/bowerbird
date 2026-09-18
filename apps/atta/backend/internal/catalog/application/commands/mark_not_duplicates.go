package commands

import (
	"context"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
)

type MarkNotDuplicatesCommand struct {
	items ports.ItemRepository
	pairs ports.NotDuplicateRepository
}

func NewMarkNotDuplicatesCommand(items ports.ItemRepository, pairs ports.NotDuplicateRepository) *MarkNotDuplicatesCommand {
	if items == nil {
		panic("item repository is required")
	}
	if pairs == nil {
		panic("not-duplicate repository is required")
	}
	return &MarkNotDuplicatesCommand{items: items, pairs: pairs}
}

func (cmd *MarkNotDuplicatesCommand) Execute(ctx context.Context, itemIDs []string) error {
	pairs, err := domain.NotDuplicatePairsFromIDs(itemIDs)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "at least two catalog item ids are required")
	}
	for _, pair := range pairs {
		for _, id := range []string{pair.Left, pair.Right} {
			item, err := cmd.items.GetItemByID(ctx, id)
			if err != nil {
				return err
			}
			if item == nil || item.IsMerged() {
				return appErrors.New(appErrors.CodeValidation, "catalog item not found")
			}
		}
	}
	return cmd.pairs.UpsertNotDuplicatePairs(ctx, pairs)
}
