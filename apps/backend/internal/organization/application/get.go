package application

import (
	"context"

	"github.com/bowerbird/internal/organization/application/ports"
	"github.com/bowerbird/internal/organization/application/queries"
	"github.com/bowerbird/internal/organization/domain"
)

type GetOrganizationUseCase struct {
	query *queries.GetOrganizationQuery
}

func NewGetOrganizationUseCase(repo ports.OrganizationRepository) *GetOrganizationUseCase {
	return &GetOrganizationUseCase{
		query: queries.NewGetOrganizationQuery(repo),
	}
}

func NewGetOrganizationUseCaseFromQuery(query *queries.GetOrganizationQuery) *GetOrganizationUseCase {
	if query == nil {
		panic("get organization query is required")
	}

	return &GetOrganizationUseCase{query: query}
}

func (uc *GetOrganizationUseCase) Execute(ctx context.Context, id, userID string) (*domain.Organization, error) {
	return uc.query.Execute(ctx, id, userID)
}
