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

type RememberDecisionCommand struct {
	items    ports.ItemRepository
	aliases  ports.AliasRepository
	memories ports.MatchMemoryRepository
	links    ports.InvoiceLineLinkRepository
	now      func() time.Time
	newID    func() string
}

func NewRememberDecisionCommand(
	items ports.ItemRepository,
	aliases ports.AliasRepository,
	memories ports.MatchMemoryRepository,
	links ports.InvoiceLineLinkRepository,
) *RememberDecisionCommand {
	return &RememberDecisionCommand{
		items:    items,
		aliases:  aliases,
		memories: memories,
		links:    links,
		now:      time.Now,
		newID:    id.NewULID,
	}
}

type RememberDecisionInput struct {
	LineID      string
	ItemID      string
	Action      string // link | never_match | create_provisional
	Remember    bool
	Lock        bool
	PartyID     string
	ItemCode    string
	Description string
}

func (cmd *RememberDecisionCommand) Execute(ctx context.Context, input RememberDecisionInput) error {
	if cmd.links == nil {
		return fmt.Errorf("invoice line link repository is required")
	}

	state, err := cmd.links.GetLineLinkState(ctx, input.LineID)
	if err != nil {
		return err
	}
	if state == nil {
		return appErrors.New(appErrors.CodeNotFound, "invoice line not found")
	}

	partyID := firstNonEmpty(input.PartyID, state.PartyID)
	itemCode := firstNonEmpty(input.ItemCode, state.ItemCode)
	description := firstNonEmpty(input.Description, state.Description)

	action := strings.TrimSpace(input.Action)
	if action == "" {
		action = domain.MemoryActionLink
	}
	if action == "create_provisional" {
		itemID, err := cmd.mintProvisionalItem(ctx, partyID, itemCode, description)
		if err != nil {
			return err
		}
		input.ItemID = itemID
		input.Action = domain.MemoryActionLink
		input.Remember = true
		if !input.Lock {
			input.Lock = true
		}
		action = domain.MemoryActionLink
	}

	decision, err := domain.DecideManualLink(input.Action, input.ItemID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidMemoryAction) {
			return appErrors.New(appErrors.CodeValidation, "invalid memory action")
		}
		if errors.Is(err, domain.ErrItemIDRequired) {
			return appErrors.New(appErrors.CodeValidation, "item_id is required for link")
		}
		return err
	}

	if decision.Status == domain.LinkStatusLinked {
		item, err := cmd.items.GetItemByID(ctx, input.ItemID)
		if err != nil {
			return err
		}
		if item == nil {
			return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
		}
		if input.Remember && domain.NormalizeItemCode(itemCode) != "" && partyID != "" {
			if err := cmd.ensureSupplierAlias(ctx, partyID, domain.NormalizeItemCode(itemCode), input.ItemID); err != nil {
				return err
			}
		}
	}

	if err := cmd.links.UpdateLineLink(ctx, input.LineID, decision.ItemID, decision.Status, decision.Method, input.Lock, nil); err != nil {
		return err
	}

	if input.Remember {
		var memItemID *string
		if decision.ItemID != nil {
			memItemID = decision.ItemID
		} else if action == domain.MemoryActionNeverMatch && strings.TrimSpace(input.ItemID) != "" {
			blocked := strings.TrimSpace(input.ItemID)
			memItemID = &blocked
		}
		mem, err := domain.NewMatchMemory(cmd.newID(), partyID, itemCode, description, action, memItemID, cmd.now())
		if err != nil {
			if errors.Is(err, domain.ErrInvalidMemoryAction) {
				return appErrors.New(appErrors.CodeValidation, "invalid memory action")
			}
			if errors.Is(err, domain.ErrItemIDRequired) {
				return appErrors.New(appErrors.CodeValidation, "item_id is required for link")
			}
			return err
		}
		if err := cmd.memories.UpsertMemory(ctx, mem); err != nil {
			return err
		}
	}

	if state.InvoiceHeaderID != "" {
		if err := cmd.links.SyncHeaderLinkingStatus(ctx, state.InvoiceHeaderID); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *RememberDecisionCommand) mintProvisionalItem(ctx context.Context, partyID, code, description string) (string, error) {
	if cmd.items == nil {
		return "", fmt.Errorf("item repository is required")
	}
	now := cmd.now().UTC()
	item, err := domain.NewProvisionalItem(cmd.newID(), description, code, now)
	if err != nil {
		if errors.Is(err, domain.ErrMissingItemName) {
			return "", appErrors.New(appErrors.CodeValidation, "description or item code is required to create a provisional item")
		}
		return "", err
	}
	if err := cmd.items.CreateItem(ctx, item); err != nil {
		return "", err
	}
	normalized := domain.NormalizeItemCode(code)
	if normalized != "" && strings.TrimSpace(partyID) != "" && cmd.aliases != nil {
		if err := cmd.ensureSupplierAlias(ctx, partyID, normalized, item.ID); err != nil {
			return "", err
		}
	}
	return item.ID, nil
}

func (cmd *RememberDecisionCommand) ensureSupplierAlias(ctx context.Context, partyID, code, itemID string) error {
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
	// Race: another writer created the same alias — accept if it points at the same item.
	again, findErr := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if findErr != nil {
		return findErr
	}
	if again != nil && again.PointsTo(itemID) {
		return nil
	}
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item")
}

type LinkInvoiceLineCommand struct {
	remember *RememberDecisionCommand
}

func NewLinkInvoiceLineCommand(remember *RememberDecisionCommand) *LinkInvoiceLineCommand {
	return &LinkInvoiceLineCommand{remember: remember}
}

func (cmd *LinkInvoiceLineCommand) Execute(ctx context.Context, input RememberDecisionInput) error {
	input.Action = domain.MemoryActionLink
	return cmd.remember.Execute(ctx, input)
}
