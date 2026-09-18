package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/atta/internal/connections/api"
	"github.com/atta/internal/connections/application/ports"
)

type GetConnectionQuery struct {
	repo ports.ConnectionRepository
}

func NewGetConnectionQuery(repo ports.ConnectionRepository) *GetConnectionQuery {
	if repo == nil {
		panic("connection repository is required")
	}

	return &GetConnectionQuery{repo: repo}
}

func (q *GetConnectionQuery) Execute(ctx context.Context, connectionID string) (ConnectionInfo, error) {
	if strings.TrimSpace(connectionID) == "" {
		return ConnectionInfo{}, api.ErrConnectionNotFound
	}

	conn, err := q.repo.GetByID(ctx, connectionID)
	if err != nil {
		return ConnectionInfo{}, fmt.Errorf("get connection %s: %w", connectionID, err)
	}
	if conn == nil {
		return ConnectionInfo{}, api.ErrConnectionNotFound
	}

	return ConnectionInfo{
		ID:                   conn.ID,
		Provider:             conn.Provider,
		ProviderAccountEmail: conn.ProviderAccountEmail,
		OwnerUserID:          conn.OwnerUserID,
		SharingPolicy:        conn.SharingPolicy,
	}, nil
}
