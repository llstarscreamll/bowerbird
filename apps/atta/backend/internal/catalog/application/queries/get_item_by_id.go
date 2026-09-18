package queries

import (
	"context"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
)

type ItemDetail struct {
	Item    domain.Item
	Aliases []domain.Alias
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

func (q *GetItemByIDQuery) Execute(ctx context.Context, id string) (*ItemDetail, error) {
	item, err := q.items.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	if item.IsMerged() {
		return nil, appErrors.New(appErrors.CodeGone, "catalog item was merged").WithMeta("merged_into_id", item.MergedIntoID)
	}
	aliases, err := q.aliases.ListAliasesByItemID(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	if aliases == nil {
		aliases = []domain.Alias{}
	}
	return &ItemDetail{Item: *item, Aliases: aliases}, nil
}
