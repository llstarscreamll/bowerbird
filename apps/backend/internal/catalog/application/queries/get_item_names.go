package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
)

type GetItemNamesQuery struct {
	repo ports.ItemRepository
}

func NewGetItemNamesQuery(repo ports.ItemRepository) *GetItemNamesQuery {
	return &GetItemNamesQuery{repo: repo}
}

func (q *GetItemNamesQuery) Execute(ctx context.Context, ids []string) (map[string]string, error) {
	return q.repo.GetItemNames(ctx, ids)
}
