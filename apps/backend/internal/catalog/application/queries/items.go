package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type GetItemByIDQuery struct {
	repo ports.ItemRepository
}

func NewGetItemByIDQuery(repo ports.ItemRepository) *GetItemByIDQuery {
	return &GetItemByIDQuery{repo: repo}
}

func (q *GetItemByIDQuery) Execute(ctx context.Context, id string) (*domain.Item, error) {
	item, err := q.repo.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return item, nil
}

type ListItemsQuery struct {
	repo ports.ItemRepository
}

func NewListItemsQuery(repo ports.ItemRepository) *ListItemsQuery {
	return &ListItemsQuery{repo: repo}
}

func (q *ListItemsQuery) Execute(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	return q.repo.ListItems(ctx, filter)
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
