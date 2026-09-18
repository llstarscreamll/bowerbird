package commands

import (
	"context"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
)

type CreateItemCommand struct {
	items ports.ItemRepository
	now   func() time.Time
}

func NewCreateItemCommand(items ports.ItemRepository) *CreateItemCommand {
	if items == nil {
		panic("item repository is required")
	}
	return &CreateItemCommand{items: items, now: time.Now}
}

type CreateItemInput struct {
	ID           string
	Name         string
	Kind         string
	InternalCode string
}

func (cmd *CreateItemCommand) Execute(ctx context.Context, input CreateItemInput) error {
	if !id.IsValidULID(input.ID) {
		return appErrors.New(appErrors.CodeValidation, "item id must be a valid ULID")
	}
	kind, err := domain.ParseItemKind(input.Kind)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "invalid item kind")
	}
	code, err := domain.ParseInternalCode(input.InternalCode)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "internal_code is required")
	}
	existing, err := cmd.items.GetItemByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
	}

	now := cmd.now().UTC()
	item, err := domain.NewManualItem(input.ID, input.Name, kind, code, now)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, err.Error())
	}
	return cmd.items.CreateItem(ctx, item)
}
