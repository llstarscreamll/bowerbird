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

	code := domain.NormalizeItemCode(input.ItemCode)
	descFP := domain.DescriptionFingerprint(input.Description)
	evidenceKind := domain.InferEvidenceKind(code, input.Description)
	evidenceKey := domain.EvidenceKey(input.PartyID, code, descFP, evidenceKind)

	if mem, err := cmd.memories.FindMemoryByEvidenceKey(ctx, evidenceKey); err != nil {
		return nil, err
	} else if mem != nil {
		if mem.IsNeverMatch() {
			return cmd.continueAfterNegativeMemory(ctx, input, code, mem.ItemID)
		}
		if mem.Action == domain.MemoryActionLink {
			if itemID := mem.LinkedItemID(); itemID != "" {
				result := domain.LinkedByMemory(itemID)
				return &result, nil
			}
		}
	}

	if domain.CanMintProvisional(input.PartyID, code) {
		alias, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, input.PartyID, code)
		if err != nil {
			return nil, err
		}
		if alias != nil {
			result := domain.LinkedByHardAlias(alias.ItemID)
			return &result, nil
		}
	}

	var suggestions []domain.Suggestion
	if cmd.matcher != nil && strings.TrimSpace(input.Description) != "" {
		soft, err := cmd.matcher.Match(ctx, input.Description)
		if err != nil {
			return nil, err
		}
		suggestions = soft
	}

	if domain.CanMintProvisional(input.PartyID, code) {
		item, minted, err := cmd.mintProvisional(ctx, input.PartyID, code, input.Description)
		if err != nil {
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

func (cmd *ResolveInvoiceLineCommand) continueAfterNegativeMemory(
	ctx context.Context,
	input domain.LineResolutionInput,
	code string,
	blockedItemID *string,
) (*domain.LineResolutionResult, error) {
	if domain.CanMintProvisional(input.PartyID, code) {
		alias, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, input.PartyID, code)
		if err != nil {
			return nil, err
		}
		if alias != nil && (blockedItemID == nil || !alias.PointsTo(*blockedItemID)) {
			result := domain.LinkedByHardAlias(alias.ItemID)
			return &result, nil
		}
		// Negative memory blocked the hard item: do not remint; soft-suggest only.
	}

	var suggestions []domain.Suggestion
	if cmd.matcher != nil && strings.TrimSpace(input.Description) != "" {
		soft, err := cmd.matcher.Match(ctx, input.Description)
		if err != nil {
			return nil, err
		}
		suggestions = domain.FilterBlockedSuggestions(soft, blockedItemID)
	}
	return &domain.LineResolutionResult{
		Status:      domain.SoftOrUnmatchedStatus(suggestions),
		Suggestions: suggestions,
	}, nil
}

func (cmd *ResolveInvoiceLineCommand) mintProvisional(ctx context.Context, partyID, code, description string) (*domain.Item, bool, error) {
	now := cmd.now().UTC()
	item, err := domain.NewProvisionalItem(cmd.newID(), description, code, now)
	if err != nil {
		return nil, false, err
	}
	alias, err := domain.NewSupplierSKUAlias(cmd.newID(), item.ID, partyID, code, now)
	if err != nil {
		return nil, false, err
	}
	if err := cmd.write.CreateItemWithAlias(ctx, item, alias); err != nil {
		if isConflict(err) {
			return cmd.loadWinnerBySupplierSKU(ctx, partyID, code)
		}
		return nil, false, fmt.Errorf("create provisional item+alias: %w", err)
	}
	return &item, true, nil
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

func isConflict(err error) bool {
	var appErr *appErrors.AppError
	return errors.As(err, &appErr) && appErr.Code == appErrors.CodeConflict
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
