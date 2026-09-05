package commands

import (
	"context"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type CreateItemCommand struct {
	items ports.ItemRepository
	write ports.CatalogWriteRepository
	now   func() time.Time
	newID func() string
}

func NewCreateItemCommand(items ports.ItemRepository, write ports.CatalogWriteRepository) *CreateItemCommand {
	return &CreateItemCommand{items: items, write: write, now: time.Now, newID: id.NewULID}
}

type CreateItemInput struct {
	ID          string
	Name        string
	Kind        string
	InternalSKU string
}

func (cmd *CreateItemCommand) Execute(ctx context.Context, input CreateItemInput) error {
	if !id.IsValidULID(input.ID) {
		return appErrors.New(appErrors.CodeValidation, "item id must be a valid ULID")
	}
	kind, err := domain.ParseItemKind(input.Kind)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "invalid item kind")
	}
	sku, err := domain.ParseInternalSKU(input.InternalSKU)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "internal_sku is required")
	}
	existing, err := cmd.items.GetItemByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
	}

	now := cmd.now().UTC()
	item, err := domain.NewManualItem(input.ID, input.Name, kind, sku, now)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, err.Error())
	}
	alias, err := domain.NewInternalSKUAlias(cmd.newID(), item.ID, sku, now)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, err.Error())
	}
	if err := cmd.write.CreateItemWithAlias(ctx, item, alias); err != nil {
		return err
	}
	return nil
}
