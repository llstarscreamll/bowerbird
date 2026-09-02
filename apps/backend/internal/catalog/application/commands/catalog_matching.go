package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
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

type MintProvisionalFromEvidenceCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	write   ports.CatalogWriteRepository
	now     func() time.Time
	newID   func() string
}

func NewMintProvisionalFromEvidenceCommand(items ports.ItemRepository, aliases ports.AliasRepository, write ports.CatalogWriteRepository) *MintProvisionalFromEvidenceCommand {
	return &MintProvisionalFromEvidenceCommand{
		items:   items,
		aliases: aliases,
		write:   write,
		now:     time.Now,
		newID:   id.NewULID,
	}
}

type MintProvisionalFromEvidenceInput struct {
	PartyID     string
	ItemCode    string
	Description string
}

func (cmd *MintProvisionalFromEvidenceCommand) Execute(ctx context.Context, input MintProvisionalFromEvidenceInput) (string, error) {
	if cmd.items == nil {
		return "", fmt.Errorf("item repository is required")
	}
	now := cmd.now().UTC()
	item, err := domain.NewProvisionalItem(cmd.newID(), input.Description, input.ItemCode, now)
	if err != nil {
		if errors.Is(err, domain.ErrMissingItemName) {
			return "", appErrors.New(appErrors.CodeValidation, "description or item code is required to create a provisional item")
		}
		return "", err
	}
	normalized := domain.NormalizeItemCode(input.ItemCode)
	if normalized != "" && strings.TrimSpace(input.PartyID) != "" && cmd.aliases != nil {
		alias, aliasErr := domain.NewSupplierSKUAlias(cmd.newID(), item.ID, input.PartyID, normalized, now)
		if aliasErr != nil {
			return "", aliasErr
		}
		if err := cmd.write.CreateItemWithAlias(ctx, item, alias); err != nil {
			if isConflict(err) {
				return cmd.loadWinnerItemIDBySupplierSKU(ctx, input.PartyID, normalized)
			}
			return "", err
		}
		return item.ID, nil
	}
	if err := cmd.items.CreateItem(ctx, item); err != nil {
		return "", err
	}
	return item.ID, nil
}

func (cmd *MintProvisionalFromEvidenceCommand) loadWinnerItemIDBySupplierSKU(ctx context.Context, partyID, code string) (string, error) {
	existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if err != nil {
		return "", err
	}
	if existing == nil {
		return "", appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
	}
	return existing.ItemID, nil
}

type EnsureSupplierAliasCommand struct {
	aliases ports.AliasRepository
	now     func() time.Time
	newID   func() string
}

func NewEnsureSupplierAliasCommand(aliases ports.AliasRepository) *EnsureSupplierAliasCommand {
	return &EnsureSupplierAliasCommand{
		aliases: aliases,
		now:     time.Now,
		newID:   id.NewULID,
	}
}

func (cmd *EnsureSupplierAliasCommand) Execute(ctx context.Context, partyID, code, itemID string) error {
	if cmd.aliases == nil {
		return fmt.Errorf("alias repository is required")
	}
	existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.PointsTo(itemID) {
			return nil
		}
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item")
	}

	alias, err := domain.NewSupplierSKUAlias(cmd.newID(), itemID, partyID, code, cmd.now())
	if err != nil {
		return err
	}
	err = cmd.aliases.CreateAlias(ctx, alias)
	if err == nil {
		return nil
	}
	if !isConflict(err) {
		return err
	}
	again, findErr := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if findErr != nil {
		return findErr
	}
	if again != nil && again.PointsTo(itemID) {
		return nil
	}
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item")
}

type RecordMatchMemoryCommand struct {
	memories ports.MatchMemoryRepository
	now      func() time.Time
	newID    func() string
}

func NewRecordMatchMemoryCommand(memories ports.MatchMemoryRepository) *RecordMatchMemoryCommand {
	return &RecordMatchMemoryCommand{
		memories: memories,
		now:      time.Now,
		newID:    id.NewULID,
	}
}

type RecordMatchMemoryInput struct {
	PartyID     string
	ItemCode    string
	Description string
	Action      string
	ItemID      *string
}

func (cmd *RecordMatchMemoryCommand) Execute(ctx context.Context, input RecordMatchMemoryInput) error {
	if cmd.memories == nil {
		return fmt.Errorf("match memory repository is required")
	}
	mem, err := domain.NewMatchMemory(cmd.newID(), input.PartyID, input.ItemCode, input.Description, input.Action, input.ItemID, cmd.now())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidMemoryAction) {
			return appErrors.New(appErrors.CodeValidation, "invalid memory action")
		}
		if errors.Is(err, domain.ErrItemIDRequired) {
			return appErrors.New(appErrors.CodeValidation, "item_id is required for link")
		}
		return err
	}
	return cmd.memories.UpsertMemory(ctx, mem)
}
