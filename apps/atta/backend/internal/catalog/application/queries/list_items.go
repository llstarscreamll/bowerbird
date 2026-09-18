package queries

import (
	"context"

	"github.com/atta/internal/catalog/application/ports"
)

type ListItemsQuery struct {
	items ports.ItemRepository
}

func NewListItemsQuery(items ports.ItemRepository) *ListItemsQuery {
	if items == nil {
		panic("item repository is required")
	}
	return &ListItemsQuery{items: items}
}

func (q *ListItemsQuery) Execute(ctx context.Context, filter ports.ItemListFilter) (ports.ItemListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return q.items.ListItems(ctx, filter)
}
