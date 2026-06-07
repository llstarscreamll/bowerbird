package queries

import (
	"context"

	"github.com/bowerbird/internal/organization/application/ports"
	"github.com/bowerbird/internal/organization/domain"
)

type GetOrganizationQuery struct {
	repo ports.OrganizationRepository
}

func NewGetOrganizationQuery(repo ports.OrganizationRepository) *GetOrganizationQuery {
	return &GetOrganizationQuery{repo: repo}
}

func (q *GetOrganizationQuery) Execute(ctx context.Context, id, userID string) (*domain.Organization, error) {
	return q.repo.GetByID(ctx, id, userID)
}
