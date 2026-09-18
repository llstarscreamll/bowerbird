package queries

import (
	"context"

	"github.com/atta/internal/legalentities/application/ports"
	"github.com/atta/internal/legalentities/domain"
)

type ListLegalEntitiesQuery struct {
	repo ports.LegalEntityRepository
}

func NewListLegalEntitiesQuery(repo ports.LegalEntityRepository) *ListLegalEntitiesQuery {
	if repo == nil {
		panic("legal entity repository is required")
	}
	return &ListLegalEntitiesQuery{repo: repo}
}

func (q *ListLegalEntitiesQuery) Execute(ctx context.Context) ([]domain.LegalEntity, error) {
	return q.repo.List(ctx)
}
