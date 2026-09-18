package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
)

type ResolveInvoiceLineCommand struct {
	items    ports.ItemRepository
	aliases  ports.AliasRepository
	write    ports.CatalogWriteRepository
	memories ports.MatchMemoryRepository
	matcher  ports.SoftMatcher
	now      func() time.Time
	newID    func() string
}

func NewResolveInvoiceLineCommand(
	items ports.ItemRepository,
	aliases ports.AliasRepository,
	write ports.CatalogWriteRepository,
	memories ports.MatchMemoryRepository,
	matcher ports.SoftMatcher,
) *ResolveInvoiceLineCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	if write == nil {
		panic("catalog write repository is required")
	}
	if memories == nil {
		panic("match memory repository is required")
	}
	if matcher == nil {
		panic("soft matcher is required")
	}
	return &ResolveInvoiceLineCommand{
		items:    items,
		aliases:  aliases,
		write:    write,
		memories: memories,
		matcher:  matcher,
		now:      time.Now,
		newID:    id.NewULID,
	}
}

func (cmd *ResolveInvoiceLineCommand) Execute(ctx context.Context, input domain.LineResolutionInput) (*domain.LineResolutionResult, error) {
	if preserved := domain.PreserveLockedLink(input); preserved != nil {
		return preserved, nil
	}

	seller := domain.UsableSellerSKU(input.SellerSKU)
	descFP := domain.DescriptionFingerprint(input.Description)
	evidenceKind := domain.InferEvidenceKind(seller, input.Description)
	evidenceKey := domain.EvidenceKey(input.PartyID, seller, descFP, evidenceKind)

	if mem, err := cmd.memories.FindMemoryByEvidenceKey(ctx, evidenceKey); err != nil {
		return nil, err
	} else if mem != nil {
		if mem.IsNeverMatch() {
			return cmd.afterNegativeMemory(ctx, input, mem.ItemID)
		}
		if mem.Action == domain.MemoryActionLink {
			if itemID := mem.LinkedItemID(); itemID != "" {
				result := domain.LinkedByMemory(itemID)
				return &result, nil
			}
		}
	}

	hits, err := cmd.collectHardHits(ctx, input)
	if err != nil {
		return nil, err
	}
	if agreed, conflict, ids := hits.Agree(); conflict {
		result := domain.LinkedByHardConflict(ids)
		return &result, nil
	} else if agreed != "" {
		result := domain.LinkedByHardAlias(agreed)
		return &result, nil
	}

	suggestions, err := cmd.softSuggestions(ctx, input.Description)
	if err != nil {
		return nil, err
	}

	if domain.CanMintProvisional(input.PartyID, seller) {
		item, minted, err := cmd.mintProvisional(ctx, input, seller)
		if err != nil {
			if isConflict(err) {
				if ids := conflictIDsFromErr(err); len(ids) > 0 {
					result := domain.LinkedByHardConflict(ids)
					return &result, nil
				}
				return cmd.loadWinnerResult(ctx, input.PartyID, seller, suggestions)
			}
			return nil, err
		}
		result := domain.LinkedByProvisionalMint(item.ID, minted, suggestions)
		return &result, nil
	}

	return &domain.LineResolutionResult{
		Status:      domain.SoftOrUnmatchedStatus(suggestions),
		Suggestions: suggestions,
	}, nil
}

func (cmd *ResolveInvoiceLineCommand) afterNegativeMemory(
	ctx context.Context,
	input domain.LineResolutionInput,
	blockedItemID *string,
) (*domain.LineResolutionResult, error) {
	hits, err := cmd.collectHardHits(ctx, input)
	if err != nil {
		return nil, err
	}
	hits = suppressBlocked(hits, blockedItemID)
	if agreed, conflict, ids := hits.Agree(); conflict {
		result := domain.LinkedByHardConflict(ids)
		return &result, nil
	} else if agreed != "" {
		result := domain.LinkedByHardAlias(agreed)
		return &result, nil
	}
	suggestions, err := cmd.softSuggestions(ctx, input.Description)
	if err != nil {
		return nil, err
	}
	suggestions = domain.FilterBlockedSuggestions(suggestions, blockedItemID)
	return &domain.LineResolutionResult{
		Status:      domain.SoftOrUnmatchedStatus(suggestions),
		Suggestions: suggestions,
	}, nil
}

func suppressBlocked(hits domain.HardHits, blocked *string) domain.HardHits {
	if blocked == nil || strings.TrimSpace(*blocked) == "" {
		return hits
	}
	id := strings.TrimSpace(*blocked)
	if hits.BuyerItemID == id {
		hits.BuyerItemID = ""
	}
	if hits.GTINItemID == id {
		hits.GTINItemID = ""
	}
	if hits.SellerItemID == id {
		hits.SellerItemID = ""
	}
	return hits
}

func (cmd *ResolveInvoiceLineCommand) collectHardHits(ctx context.Context, input domain.LineResolutionInput) (domain.HardHits, error) {
	hits := domain.HardHits{}
	if buyer := strings.TrimSpace(input.BuyerCode); buyer != "" {
		items, err := cmd.items.GetItemsByInternalCodes(ctx, []string{buyer})
		if err != nil {
			return hits, err
		}
		if len(items) == 1 {
			hits.BuyerItemID = items[0].ID
		}
	}
	if gtin := strings.TrimSpace(input.GTIN); gtin != "" {
		alias, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeGTIN, "", gtin)
		if err != nil {
			return hits, err
		}
		if alias != nil {
			hits.GTINItemID = alias.ItemID
		}
	}
	seller := domain.UsableSellerSKU(input.SellerSKU)
	if seller != "" && strings.TrimSpace(input.PartyID) != "" {
		alias, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, input.PartyID, seller)
		if err != nil {
			return hits, err
		}
		if alias != nil {
			hits.SellerItemID = alias.ItemID
		}
	}
	return hits, nil
}

func (cmd *ResolveInvoiceLineCommand) softSuggestions(ctx context.Context, description string) ([]domain.Suggestion, error) {
	if strings.TrimSpace(description) == "" {
		return nil, nil
	}
	return cmd.matcher.Match(ctx, description)
}

func (cmd *ResolveInvoiceLineCommand) mintProvisional(ctx context.Context, input domain.LineResolutionInput, seller string) (*domain.Item, bool, error) {
	now := cmd.now().UTC()
	item, err := domain.NewProvisionalItem(cmd.newID(), input.Description, seller, now)
	if err != nil {
		return nil, false, err
	}
	aliases := make([]domain.Alias, 0, 2)
	sku, err := domain.NewSupplierSKUAlias(cmd.newID(), item.ID, input.PartyID, seller, domain.AliasSourceInvoice, now)
	if err != nil {
		return nil, false, err
	}
	aliases = append(aliases, sku)
	if gtin := strings.TrimSpace(input.GTIN); gtin != "" {
		existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeGTIN, "", gtin)
		if err != nil {
			return nil, false, err
		}
		if existing != nil && !existing.PointsTo(item.ID) {
			return nil, false, aliasOwnedBy(existing.ItemID)
		}
		if existing == nil {
			g, err := domain.NewGTINAlias(cmd.newID(), item.ID, gtin, domain.AliasSourceInvoice, now)
			if err != nil {
				return nil, false, err
			}
			aliases = append(aliases, g)
		}
	}
	if err := cmd.write.CreateItemWithAliases(ctx, item, aliases); err != nil {
		if isConflict(err) {
			return cmd.loadWinnerBySupplierSKU(ctx, input.PartyID, seller)
		}
		return nil, false, fmt.Errorf("create provisional item+aliases: %w", err)
	}
	return &item, true, nil
}

func (cmd *ResolveInvoiceLineCommand) loadWinnerResult(ctx context.Context, partyID, seller string, suggestions []domain.Suggestion) (*domain.LineResolutionResult, error) {
	item, minted, err := cmd.loadWinnerBySupplierSKU(ctx, partyID, seller)
	if err != nil {
		return nil, err
	}
	result := domain.LinkedByProvisionalMint(item.ID, minted, suggestions)
	return &result, nil
}

func (cmd *ResolveInvoiceLineCommand) loadWinnerBySupplierSKU(ctx context.Context, partyID, code string) (*domain.Item, bool, error) {
	existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, fmt.Errorf("create provisional item+alias: alias conflict but winner not found")
	}
	won, err := cmd.items.GetItemByID(ctx, existing.ItemID)
	if err != nil {
		return nil, false, err
	}
	if won == nil {
		return nil, false, fmt.Errorf("create provisional item+alias: alias points to missing item %s", existing.ItemID)
	}
	return won, false, nil
}

func aliasOwnedBy(itemID string) error {
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item").
		WithMeta("item_id", itemID)
}

func conflictIDsFromErr(err error) []string {
	var appErr *appErrors.AppError
	if errors.As(err, &appErr) && appErr.Meta != nil {
		if id, ok := appErr.Meta["item_id"].(string); ok && id != "" {
			return []string{id}
		}
	}
	return nil
}

func isConflict(err error) bool {
	var appErr *appErrors.AppError
	return errors.As(err, &appErr) && appErr.Code == appErrors.CodeConflict
}
