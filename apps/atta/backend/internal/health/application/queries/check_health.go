package queries

import (
	"context"

	"github.com/atta/internal/health/application/ports"
	"github.com/atta/internal/health/domain"
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
