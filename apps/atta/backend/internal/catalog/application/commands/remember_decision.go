package commands

import (
	"context"
	"strings"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
)

type RememberDecisionCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	write   ports.CatalogWriteRepository
	now     func() time.Time
	newID   func() string
}

func NewRememberDecisionCommand(items ports.ItemRepository, aliases ports.AliasRepository, write ports.CatalogWriteRepository) *RememberDecisionCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	if write == nil {
		panic("catalog write repository is required")
	}
	return &RememberDecisionCommand{items: items, aliases: aliases, write: write, now: time.Now, newID: id.NewULID}
}

type RememberDecisionInput struct {
	PartyID     string
	SellerSKU   string
	GTIN        string
	Description string
	Action      string
	ItemID      string
}

func (cmd *RememberDecisionCommand) Execute(ctx context.Context, input RememberDecisionInput) error {
	action := strings.TrimSpace(input.Action)
	if action == "" {
		action = domain.MemoryActionLink
	}
	now := cmd.now().UTC()
	seller := domain.UsableSellerSKU(input.SellerSKU)
	aliases := make([]domain.Alias, 0, 2)

	if action == domain.MemoryActionLink {
		item, err := cmd.items.GetItemByID(ctx, input.ItemID)
		if err != nil {
			return err
		}
		if item == nil {
			return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
		}
		if seller != "" && strings.TrimSpace(input.PartyID) != "" {
			sku, err := domain.NewSupplierSKUAlias(cmd.newID(), item.ID, input.PartyID, seller, domain.AliasSourceInvoice, now)
			if err != nil {
				return err
			}
			if err := cmd.ensureAliasOwner(ctx, sku, item.ID); err != nil {
				return err
			}
			aliases = append(aliases, sku)
		}
		if gtin := strings.TrimSpace(input.GTIN); gtin != "" {
			g, err := domain.NewGTINAlias(cmd.newID(), item.ID, gtin, domain.AliasSourceInvoice, now)
			if err != nil {
				return appErrors.New(appErrors.CodeValidation, err.Error())
			}
			if err := cmd.ensureAliasOwner(ctx, g, item.ID); err != nil {
				return err
			}
			aliases = append(aliases, g)
		}
	}

	var memItemID *string
	if strings.TrimSpace(input.ItemID) != "" {
		idCopy := strings.TrimSpace(input.ItemID)
		memItemID = &idCopy
	}
	memory, err := domain.NewMatchMemory(cmd.newID(), input.PartyID, seller, input.Description, action, memItemID, now)
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, err.Error())
	}
	return cmd.write.RememberDecision(ctx, aliases, memory)
}

func (cmd *RememberDecisionCommand) ensureAliasOwner(ctx context.Context, alias domain.Alias, itemID string) error {
	existing, err := cmd.aliases.FindBySchemePartyValue(ctx, alias.Scheme, partyIDValue(alias), alias.Value)
	if err != nil {
		return err
	}
	if existing != nil && !existing.PointsTo(itemID) {
		return aliasOwnedBy(existing.ItemID)
	}
	if existing != nil {
		return nil
	}
	return nil
}
