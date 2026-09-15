package commands

import (
	"context"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type UpdateItemCommand struct {
	items ports.ItemRepository
	now   func() time.Time
}

func NewUpdateItemCommand(items ports.ItemRepository) *UpdateItemCommand {
	if items == nil {
		panic("item repository is required")
	}
	return &UpdateItemCommand{items: items, now: time.Now}
}

type UpdateItemInput struct {
	ID           string
	Name         *string
	Kind         *string
	Status       *string
	InternalCode *string
}

func (cmd *UpdateItemCommand) Execute(ctx context.Context, input UpdateItemInput) error {
	item, err := cmd.items.GetItemByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}

	now := cmd.now().UTC()

	if input.Name != nil {
		if err := item.Rename(*input.Name, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}
	if input.Kind != nil {
		kind, err := domain.ParseItemKind(*input.Kind)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "invalid item kind")
		}
		item.ChangeKind(kind, now)
	}

	var newCode *domain.InternalCode
	if input.InternalCode != nil {
		parsed, err := domain.ParseInternalCode(*input.InternalCode)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "internal_code is required")
		}
		newCode = &parsed
	}

	confirmRequested := false
	if input.Status != nil {
		var err error
		confirmRequested, err = item.InterpretMasterStatusChange(*input.Status)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}

	if confirmRequested {
		if err := item.Confirm(newCode, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	} else if newCode != nil {
		if err := item.AssignInternalCode(*newCode, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}

	return cmd.items.UpdateItem(ctx, *item)
}
