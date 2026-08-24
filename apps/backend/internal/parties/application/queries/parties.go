package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type GetPartyByIDQuery struct {
	repo ports.PartyRepository
}

func NewGetPartyByIDQuery(repo ports.PartyRepository) *GetPartyByIDQuery {
	return &GetPartyByIDQuery{repo: repo}
}

func (q *GetPartyByIDQuery) Execute(ctx context.Context, id string) (*domain.Party, error) {
	party, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if party == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "party not found")
	}
	return party, nil
}

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
