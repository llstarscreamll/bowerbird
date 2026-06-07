package queries

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/connections/application/ports"
)

type ConnectionInfo struct {
	ID                   string
	Provider             string
	ProviderAccountEmail string
	OwnerUserID          string
	SharingPolicy        string
}

type GetActiveConnectionsQuery struct {
	repo ports.ConnectionRepository
}

func NewGetActiveConnectionsQuery(repo ports.ConnectionRepository) *GetActiveConnectionsQuery {
	if repo == nil {
		panic("connection repository is required")
	}

	return &GetActiveConnectionsQuery{repo: repo}
}

func (q *GetActiveConnectionsQuery) Execute(ctx context.Context) ([]ConnectionInfo, error) {
	connections, err := q.repo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active connections: %w", err)
	}

	result := make([]ConnectionInfo, 0, len(connections))
	for _, c := range connections {
		result = append(result, ConnectionInfo{
			ID:                   c.ID,
			Provider:             c.Provider,
			ProviderAccountEmail: c.ProviderAccountEmail,
			OwnerUserID:          c.OwnerUserID,
			SharingPolicy:        c.SharingPolicy,
		})
	}

	return result, nil
}
