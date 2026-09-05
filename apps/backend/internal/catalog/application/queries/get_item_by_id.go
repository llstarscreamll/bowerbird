package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

// ItemView is the HTTP/read-model projection of a catalog item plus canonical SKU.
type ItemView struct {
	Item        domain.Item
	InternalSKU *string
}

type GetItemByIDQuery struct {
	items   ports.ItemRepository
	aliases ports.AliasRepository
}

func NewGetItemByIDQuery(items ports.ItemRepository, aliases ports.AliasRepository) *GetItemByIDQuery {
	if items == nil {
		panic("item repository is required")
	}
	if aliases == nil {
		panic("alias repository is required")
	}
	return &GetItemByIDQuery{items: items, aliases: aliases}
}

func (q *GetItemByIDQuery) Execute(ctx context.Context, id string) (*ItemView, error) {
	item, err := q.items.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	skus, err := q.aliases.ListInternalSKUsByItemIDs(ctx, []string{item.ID})
	if err != nil {
		return nil, err
	}
	view := &ItemView{Item: *item}
	if sku, ok := skus[item.ID]; ok && sku != "" {
		s := sku
		view.InternalSKU = &s
	}
	return view, nil
}
