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

type AddItemAliasCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	now     func() time.Time
}

func NewAddItemAliasCommand(items ports.ItemRepository, aliases ports.AliasRepository) *AddItemAliasCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	return &AddItemAliasCommand{items: items, aliases: aliases, now: time.Now}
}

type AddItemAliasInput struct {
	ItemID  string
	AliasID string
	Scheme  string
	Value   string
	PartyID string
}

func (cmd *AddItemAliasCommand) Execute(ctx context.Context, input AddItemAliasInput) error {
	if !id.IsValidULID(input.AliasID) {
		return appErrors.New(appErrors.CodeValidation, "alias id must be a valid ULID")
	}
	item, err := cmd.items.GetItemByID(ctx, input.ItemID)
	if err != nil {
		return err
	}
	if item == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	now := cmd.now().UTC()
	var alias domain.Alias
	switch strings.TrimSpace(input.Scheme) {
	case domain.AliasSchemeSupplierSKU:
		alias, err = domain.NewSupplierSKUAlias(input.AliasID, item.ID, input.PartyID, input.Value, domain.AliasSourceManual, now)
	case domain.AliasSchemeGTIN:
		alias, err = domain.NewGTINAlias(input.AliasID, item.ID, input.Value, domain.AliasSourceManual, now)
	default:
		return appErrors.New(appErrors.CodeValidation, "invalid alias scheme")
	}
	if err != nil {
		switch {
		case err == domain.ErrMissingAliasValue, err == domain.ErrMissingAliasParty, err == domain.ErrInvalidGTIN, err == domain.ErrInvalidAliasSource:
			return appErrors.New(appErrors.CodeValidation, err.Error())
		default:
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}
	if existing, findErr := cmd.aliases.FindBySchemePartyValue(ctx, alias.Scheme, partyIDValue(alias), alias.Value); findErr != nil {
		return findErr
	} else if existing != nil && !existing.PointsTo(item.ID) {
		return aliasOwnedBy(existing.ItemID)
	} else if existing != nil {
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists").
			WithMeta("item_id", existing.ItemID)
	}
	if err := cmd.aliases.CreateAlias(ctx, alias); err != nil {
		if isConflict(err) {
			if existing, findErr := cmd.aliases.FindBySchemePartyValue(ctx, alias.Scheme, partyIDValue(alias), alias.Value); findErr == nil && existing != nil {
				return aliasOwnedBy(existing.ItemID)
			}
		}
		return err
	}
	return nil
}

func partyIDValue(alias domain.Alias) string {
	if alias.PartyID == nil {
		return ""
	}
	return *alias.PartyID
}
