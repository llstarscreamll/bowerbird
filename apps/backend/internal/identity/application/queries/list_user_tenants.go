package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/identity/application/ports"
)

type TenantMembershipDTO struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type ListUserTenantsQuery struct {
	repo ports.Repository
}

func NewListUserTenantsQuery(repo ports.Repository) *ListUserTenantsQuery {
	return &ListUserTenantsQuery{repo: repo}
}

func (q *ListUserTenantsQuery) Execute(ctx context.Context, userID string) ([]TenantMembershipDTO, error) {
	memberships, err := q.repo.FindTenantMemberships(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}

	dtos := make([]TenantMembershipDTO, len(memberships))
	for i, m := range memberships {
		dtos[i] = TenantMembershipDTO{
			TenantID: m.TenantID,
			Name:     m.Name,
			Role:     string(m.Role),
		}
	}

	return dtos, nil
}
