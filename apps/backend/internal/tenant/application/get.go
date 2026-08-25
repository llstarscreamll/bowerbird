package application

import (
	"context"

	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/application/queries"
	"github.com/bowerbird/internal/tenant/domain"
)

type GetTenantUseCase struct {
	query *queries.GetTenantQuery
}

func NewGetTenantUseCase(repo ports.TenantRepository) *GetTenantUseCase {
	return &GetTenantUseCase{
		query: queries.NewGetTenantQuery(repo),
	}
}

func NewGetTenantUseCaseFromQuery(query *queries.GetTenantQuery) *GetTenantUseCase {
	if query == nil {
		panic("get tenant query is required")
	}

	return &GetTenantUseCase{query: query}
}

func (uc *GetTenantUseCase) Execute(ctx context.Context, id, userID string) (*domain.Tenant, error) {
	return uc.query.Execute(ctx, id, userID)
}
