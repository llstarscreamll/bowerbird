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
