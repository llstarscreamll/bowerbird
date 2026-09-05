package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
)

// ItemDisplay is name + canonical internal SKU for linked-line enrichment.
type ItemDisplay struct {
	Name        string
	InternalSKU string
}

type GetItemDisplaysQuery struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
}

func NewGetItemDisplaysQuery(items ports.ItemRepository, aliases ports.AliasRepository) *GetItemDisplaysQuery {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	return &GetItemDisplaysQuery{items: items, aliases: aliases}
}

func (q *GetItemDisplaysQuery) Execute(ctx context.Context, ids []string) (map[string]ItemDisplay, error) {
	out := make(map[string]ItemDisplay, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	names, err := q.items.GetItemNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	skus, err := q.aliases.ListInternalSKUsByItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		out[id] = ItemDisplay{
			Name:        names[id],
			InternalSKU: skus[id],
		}
	}
	return out, nil
}
