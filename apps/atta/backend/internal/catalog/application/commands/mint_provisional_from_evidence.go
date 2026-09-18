package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
)

type MintProvisionalFromEvidenceCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	write   ports.CatalogWriteRepository
	now     func() time.Time
	newID   func() string
}

func NewMintProvisionalFromEvidenceCommand(items ports.ItemRepository, aliases ports.AliasRepository, write ports.CatalogWriteRepository) *MintProvisionalFromEvidenceCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	if write == nil {
		panic("catalog write repository is required")
	}
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
	SellerSKU   string
	GTIN        string
	Description string
}

func (cmd *MintProvisionalFromEvidenceCommand) Execute(ctx context.Context, input MintProvisionalFromEvidenceInput) (string, error) {
	seller := domain.UsableSellerSKU(input.SellerSKU)
	if !domain.CanMintProvisional(input.PartyID, seller) {
		return "", appErrors.New(appErrors.CodeValidation, "usable seller sku and party are required to mint a provisional item")
	}
	now := cmd.now().UTC()
	item, err := domain.NewProvisionalItem(cmd.newID(), input.Description, seller, now)
	if err != nil {
		if errors.Is(err, domain.ErrMissingItemName) {
			return "", appErrors.New(appErrors.CodeValidation, "description or item code is required to create a provisional item")
		}
		return "", err
	}
	aliases := make([]domain.Alias, 0, 2)
	sku, err := domain.NewSupplierSKUAlias(cmd.newID(), item.ID, input.PartyID, seller, domain.AliasSourceInvoice, now)
	if err != nil {
		return "", err
	}
	aliases = append(aliases, sku)
	if gtin := strings.TrimSpace(input.GTIN); gtin != "" {
		existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeGTIN, "", gtin)
		if err != nil {
			return "", err
		}
		if existing != nil && !existing.PointsTo(item.ID) {
			return "", aliasOwnedBy(existing.ItemID)
		}
		if existing == nil {
			g, err := domain.NewGTINAlias(cmd.newID(), item.ID, gtin, domain.AliasSourceInvoice, now)
			if err != nil {
				return "", appErrors.New(appErrors.CodeValidation, err.Error())
			}
			aliases = append(aliases, g)
		}
	}
	if err := cmd.write.CreateItemWithAliases(ctx, item, aliases); err != nil {
		if isConflict(err) {
			return cmd.loadWinnerItemIDBySupplierSKU(ctx, input.PartyID, seller)
		}
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
