package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/organization/domain"
)

type GetOrganizationUseCase struct {
	repo domain.Repository
}

func NewGetOrganizationUseCase(repo domain.Repository) *GetOrganizationUseCase {
	return &GetOrganizationUseCase{
		repo: repo,
	}
}

func (uc *GetOrganizationUseCase) Execute(ctx context.Context, id, userID string) (*domain.Organization, error) {
	return uc.repo.GetByID(ctx, id, userID)
}
