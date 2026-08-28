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

type GetItemNamesQuery struct {
	repo ports.ItemRepository
}

func NewGetItemNamesQuery(repo ports.ItemRepository) *GetItemNamesQuery {
	return &GetItemNamesQuery{repo: repo}
}

func (q *GetItemNamesQuery) Execute(ctx context.Context, ids []string) (map[string]string, error) {
	return q.repo.GetItemNames(ctx, ids)
}

type ListReviewQueueQuery struct {
	repo ports.InvoiceLineLinkRepository
}

func NewListReviewQueueQuery(repo ports.InvoiceLineLinkRepository) *ListReviewQueueQuery {
	return &ListReviewQueueQuery{repo: repo}
}

func (q *ListReviewQueueQuery) Execute(ctx context.Context) ([]ports.ReviewLine, error) {
	return q.repo.ListReviewLines(ctx, []string{domain.LinkStatusUnmatched, domain.LinkStatusSuggested})
}
