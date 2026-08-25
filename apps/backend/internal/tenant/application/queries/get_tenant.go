package queries

import (
	"context"

	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/domain"
)

type GetTenantQuery struct {
	repo ports.TenantRepository
}

func NewGetTenantQuery(repo ports.TenantRepository) *GetTenantQuery {
	return &GetTenantQuery{repo: repo}
}

func (q *GetTenantQuery) Execute(ctx context.Context, id, userID string) (*domain.Tenant, error) {
	return q.repo.GetByID(ctx, id, userID)
}
