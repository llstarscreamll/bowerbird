package queries

import (
	"context"

	"github.com/bowerbird/internal/connections/application"
	"github.com/bowerbird/internal/inbox/domain"
)

type AccountSyncStatus struct {
	ID           string  `json:"id"`
	Provider     string  `json:"provider"`
	EmailAddress string  `json:"email_address"`
	Status       string  `json:"status"` // the sync status
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
}

type ListAccountHealthQuery struct {
	repo               domain.SyncCursorRepository
	connectionsService application.InternalService
}

func NewListAccountHealthQuery(repo domain.SyncCursorRepository, connectionsService application.InternalService) *ListAccountHealthQuery {
	return &ListAccountHealthQuery{repo: repo, connectionsService: connectionsService}
}

func (q *ListAccountHealthQuery) Execute(ctx context.Context) ([]AccountSyncStatus, error) {
	connections, err := q.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]AccountSyncStatus, 0, len(connections))
	for _, conn := range connections {
		cursor, err := q.repo.GetSyncCursor(ctx, conn.ID)
		if err != nil {
			return nil, err
		}

		var lastSyncedAt *string
		status := domain.SyncCursorStatusIdle.String()

		if cursor != nil {
			status = cursor.Status.String()
			if cursor.LastSyncedAt != nil {
				t := cursor.LastSyncedAt.Format("2006-01-02T15:04:05Z07:00")
				lastSyncedAt = &t
			}
		}

		summaries = append(summaries, AccountSyncStatus{
			ID:           conn.ID,
			Provider:     conn.Provider,
			EmailAddress: conn.ProviderAccountEmail,
			Status:       status,
			LastSyncedAt: lastSyncedAt,
		})
	}

	return summaries, nil
}
