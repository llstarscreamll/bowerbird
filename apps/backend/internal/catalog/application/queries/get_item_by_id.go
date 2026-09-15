package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type GetItemByIDQuery struct {
	items ports.ItemRepository
}

func NewGetItemByIDQuery(items ports.ItemRepository) *GetItemByIDQuery {
	if items == nil {
		panic("item repository is required")
	}
	return &GetItemByIDQuery{items: items}
}

func (q *GetItemByIDQuery) Execute(ctx context.Context, id string) (*domain.Item, error) {
	item, err := q.items.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return item, nil
}
