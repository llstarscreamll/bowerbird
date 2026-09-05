package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
)

type ListItemsQuery struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
}

func NewListItemsQuery(items ports.ItemRepository, aliases ports.AliasRepository) *ListItemsQuery {
	return &ListItemsQuery{items: items, aliases: aliases}
}

func (q *ListItemsQuery) Execute(ctx context.Context, filter ports.ItemListFilter) ([]ItemView, error) {
	items, err := q.items.ListItems(ctx, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	skus, err := q.aliases.ListInternalSKUsByItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ItemView, 0, len(items))
	for _, item := range items {
		view := ItemView{Item: item}
		if sku, ok := skus[item.ID]; ok && sku != "" {
			s := sku
			view.InternalSKU = &s
		}
		out = append(out, view)
	}
	return out, nil
}
