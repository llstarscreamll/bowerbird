package commands

import (
	"context"
	"strings"
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

type UpdateItemCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	write   ports.CatalogWriteRepository
	now     func() time.Time
	newID   func() string
}

func NewUpdateItemCommand(items ports.ItemRepository, aliases ports.AliasRepository, write ports.CatalogWriteRepository) *UpdateItemCommand {
	return &UpdateItemCommand{items: items, aliases: aliases, write: write, now: time.Now, newID: id.NewULID}
}

type UpdateItemInput struct {
	ID          string
	Name        *string
	Kind        *string
	Status      *string
	InternalSKU *string
}

func (cmd *UpdateItemCommand) Execute(ctx context.Context, input UpdateItemInput) error {
	item, err := cmd.items.GetItemByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}

	skus, err := cmd.aliases.ListInternalSKUsByItemIDs(ctx, []string{item.ID})
	if err != nil {
		return err
	}
	var currentSKU *domain.InternalSKU
	if raw, ok := skus[item.ID]; ok && strings.TrimSpace(raw) != "" {
		parsed, parseErr := domain.ParseInternalSKU(raw)
		if parseErr != nil {
			return parseErr
		}
		currentSKU = &parsed
	}

	now := cmd.now().UTC()
	var newAlias *domain.Alias

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

	var newSKU *domain.InternalSKU
	if input.InternalSKU != nil {
		parsed, err := domain.ParseInternalSKU(*input.InternalSKU)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "internal_sku is required")
		}
		newSKU = &parsed
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
		sku, assignNew, err := item.Confirm(currentSKU, newSKU, now)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
		if assignNew {
			alias, err := domain.NewInternalSKUAlias(cmd.newID(), item.ID, sku, now)
			if err != nil {
				return appErrors.New(appErrors.CodeValidation, err.Error())
			}
			newAlias = &alias
		}
	} else if newSKU != nil {
		assignNew, err := item.AssignInternalSKU(currentSKU, *newSKU, now)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
		if assignNew {
			alias, err := domain.NewInternalSKUAlias(cmd.newID(), item.ID, *newSKU, now)
			if err != nil {
				return appErrors.New(appErrors.CodeValidation, err.Error())
			}
			newAlias = &alias
		}
	}

	return cmd.write.UpdateItemWithOptionalAlias(ctx, *item, newAlias)
}
