package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
)

type ListPartiesQuery struct {
	repo ports.PartyRepository
}

func NewListPartiesQuery(repo ports.PartyRepository) *ListPartiesQuery {
	return &ListPartiesQuery{repo: repo}
}

func (q *ListPartiesQuery) Execute(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	if q.repo == nil {
		return nil, fmt.Errorf("party repository is required")
	}
	return q.repo.List(ctx, filter)
}
