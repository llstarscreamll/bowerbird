package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/health/domain"
)

type CheckHealthUseCase struct {
	repo domain.Repository
}

func NewCheckHealthUseCase(repo domain.Repository) CheckHealthUseCase {
	return CheckHealthUseCase{repo: repo}
}

func (uc CheckHealthUseCase) Execute(ctx context.Context) domain.Health {
	if err := uc.repo.Ping(ctx); err != nil {
		return domain.Health{Status: domain.StatusDegraded}
	}

	return domain.Health{Status: domain.StatusOK}
}
