package queries

import (
	"context"

	"github.com/bowerbird/internal/health/application/ports"
	"github.com/bowerbird/internal/health/domain"
)

type CheckHealthQuery struct {
	repo ports.HealthRepository
}

func NewCheckHealthQuery(repo ports.HealthRepository) *CheckHealthQuery {
	if repo == nil {
		panic("repo is required")
	}
	return &CheckHealthQuery{repo: repo}
}

func (q *CheckHealthQuery) Execute(ctx context.Context) domain.Health {
	if err := q.repo.Ping(ctx); err != nil {
		return domain.Health{Status: domain.StatusDegraded}
	}

	return domain.Health{Status: domain.StatusOK}
}
