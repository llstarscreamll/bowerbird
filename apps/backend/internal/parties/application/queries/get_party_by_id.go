package queries

import (
	"context"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type GetPartyByIDQuery struct {
	repo ports.PartyRepository
}

func NewGetPartyByIDQuery(repo ports.PartyRepository) *GetPartyByIDQuery {
	if repo == nil {
		panic("party repository is required")
	}
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
