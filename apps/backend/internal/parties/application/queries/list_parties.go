package queries

import (
	"context"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
)

type ListPartiesQuery struct {
	repo ports.PartyRepository
}

func NewListPartiesQuery(repo ports.PartyRepository) *ListPartiesQuery {
	if repo == nil {
		panic("party repository is required")
	}
	return &ListPartiesQuery{repo: repo}
}

func (q *ListPartiesQuery) Execute(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	return q.repo.List(ctx, filter)
}
