package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
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

func (q *ListItemsQuery) Execute(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	return q.items.ListItems(ctx, filter)
}
