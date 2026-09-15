package commands

import (
	"context"
	"strings"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type MergeItemsCommand struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
	merge   ports.ItemMergeRepository
	links   ports.ItemLinkSupport
	now     func() time.Time
}

func NewMergeItemsCommand(
	items ports.ItemRepository,
	aliases ports.AliasRepository,
	merge ports.ItemMergeRepository,
	links ports.ItemLinkSupport,
) *MergeItemsCommand {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	if merge == nil {
		panic("item merge repository is required")
	}
	return &MergeItemsCommand{items: items, aliases: aliases, merge: merge, links: links, now: time.Now}
}

func (cmd *MergeItemsCommand) BindLinks(links ports.ItemLinkSupport) {
	cmd.links = links
}

type MergeItemsInput struct {
	SurvivorID   string
	SourceIDs    []string
	Name         *string
	Kind         *string
	InternalCode *string
}

func (cmd *MergeItemsCommand) Execute(ctx context.Context, input MergeItemsInput) error {
	if cmd.links == nil {
		return appErrors.New(appErrors.CodeInternal, "item link support is not bound")
	}
	survivorID := strings.TrimSpace(input.SurvivorID)
	if survivorID == "" {
		return appErrors.New(appErrors.CodeValidation, "survivor_id is required")
	}
	sourceIDs := uniqueTrimmed(input.SourceIDs)
	if len(sourceIDs) < 1 {
		return appErrors.New(appErrors.CodeValidation, "source_ids is required")
	}
	filtered := make([]string, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		if id == survivorID {
			continue
		}
		filtered = append(filtered, id)
	}
	if len(filtered) < 1 {
		return appErrors.New(appErrors.CodeValidation, "source_ids is required")
	}
	if 1+len(filtered) > 5 {
		return appErrors.New(appErrors.CodeValidation, "merge accepts at most 5 items")
	}

	survivor, err := cmd.items.GetItemByID(ctx, survivorID)
	if err != nil {
		return err
	}
	if survivor == nil {
		return appErrors.New(appErrors.CodeValidation, "survivor catalog item not found")
	}
	if survivor.IsMerged() {
		return appErrors.New(appErrors.CodeValidation, "survivor catalog item was merged")
	}

	now := cmd.now().UTC()
	sources := make([]domain.Item, 0, len(filtered))
	fromIDs := make([]string, 0, len(filtered))
	sourceCodes := make([]string, 0)
	seenCodes := map[string]struct{}{}
	for _, id := range filtered {
		item, err := cmd.items.GetItemByID(ctx, id)
		if err != nil {
			return err
		}
		if item == nil {
			return appErrors.New(appErrors.CodeValidation, "source catalog item not found")
		}
		if item.IsMerged() && item.MergedIntoID != survivorID {
			return appErrors.New(appErrors.CodeValidation, domain.ErrItemAlreadyMerged.Error())
		}
		if code, ok := item.ParsedInternalCode(); ok {
			if _, dup := seenCodes[code.String()]; !dup {
				seenCodes[code.String()] = struct{}{}
				sourceCodes = append(sourceCodes, code.String())
			}
		}
		if err := item.MergeInto(survivorID, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
		sources = append(sources, *item)
		fromIDs = append(fromIDs, id)
	}

	if input.Name != nil {
		if err := survivor.Rename(*input.Name, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	} else if name := domain.PickMergeName(*survivor, sources); name != survivor.Name {
		if err := survivor.Rename(name, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}
	if input.Kind != nil {
		kind, err := domain.ParseItemKind(*input.Kind)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "invalid item kind")
		}
		if err := survivor.ChangeKind(kind, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	} else if kindValue := domain.PickMergeKind(*survivor, sources); kindValue != survivor.Kind {
		kind, err := domain.ParseItemKind(kindValue)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "invalid item kind")
		}
		if err := survivor.ChangeKind(kind, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}
	if err := cmd.applyInternalCode(survivor, sourceCodes, input.InternalCode, now); err != nil {
		return err
	}

	deleteIDs, reassign, err := cmd.unionAliases(ctx, survivorID, fromIDs)
	if err != nil {
		return err
	}

	if err := cmd.merge.ApplyMerge(ctx, ports.MergePersistence{
		Survivor:        *survivor,
		Merged:          sources,
		ReassignAliases: reassign,
		DeleteAliasIDs:  deleteIDs,
	}); err != nil {
		return err
	}
	if err := cmd.links.RelinkItems(ctx, fromIDs, survivorID); err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "invoice line relink failed after catalog merge")
	}
	return nil
}

func (cmd *MergeItemsCommand) applyInternalCode(survivor *domain.Item, sourceCodes []string, chosen *string, now time.Time) error {
	_, survivorHas := survivor.ParsedInternalCode()
	if chosen != nil {
		code, err := domain.ParseInternalCode(*chosen)
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "internal_code is required")
		}
		if err := survivor.AssignInternalCode(code, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
		return nil
	}
	if survivorHas {
		return nil
	}
	if len(sourceCodes) > 1 {
		return appErrors.New(appErrors.CodeValidation, "internal_code must be chosen when items have different internal codes")
	}
	if len(sourceCodes) == 1 {
		code, err := domain.ParseInternalCode(sourceCodes[0])
		if err != nil {
			return appErrors.New(appErrors.CodeValidation, "internal_code is required")
		}
		if err := survivor.AssignInternalCode(code, now); err != nil {
			return appErrors.New(appErrors.CodeValidation, err.Error())
		}
	}
	return nil
}

func (cmd *MergeItemsCommand) unionAliases(ctx context.Context, survivorID string, sourceIDs []string) ([]string, []domain.Alias, error) {
	held, err := cmd.aliases.ListAliasesByItemID(ctx, survivorID)
	if err != nil {
		return nil, nil, err
	}
	var incoming []domain.Alias
	for _, id := range sourceIDs {
		aliases, err := cmd.aliases.ListAliasesByItemID(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		incoming = append(incoming, aliases...)
	}
	reassign, deleteIDs, err := domain.UnionAliases(survivorID, held, incoming)
	if err != nil {
		return nil, nil, appErrors.New(appErrors.CodeValidation, err.Error())
	}
	return deleteIDs, reassign, nil
}

func uniqueTrimmed(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
