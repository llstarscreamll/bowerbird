package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/connections/application/ports"
)

type GetSharingPolicyQuery struct {
	repo ports.ConnectionRepository
}

func NewGetSharingPolicyQuery(repo ports.ConnectionRepository) *GetSharingPolicyQuery {
	if repo == nil {
		panic("connection repository is required")
	}

	return &GetSharingPolicyQuery{repo: repo}
}

func (q *GetSharingPolicyQuery) Execute(ctx context.Context, connectionID string) (string, error) {
	conn, err := q.repo.GetByID(ctx, connectionID)
	if err != nil {
		return "", fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return "", fmt.Errorf("connection not found")
	}

	return conn.SharingPolicy, nil
}
