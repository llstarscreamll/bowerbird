package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type AccountSyncStatus struct {
	ID           string  `json:"id"`
	Provider     string  `json:"provider"`
	EmailAddress string  `json:"email_address"`
	Status       string  `json:"status"` // the sync status
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
}

type ListAccountHealthUseCase struct {
	repo               domain.Repository
	connectionsService application.InternalService
}

func NewListAccountHealthUseCase(repo domain.Repository, connectionsService application.InternalService) *ListAccountHealthUseCase {
	return &ListAccountHealthUseCase{repo: repo, connectionsService: connectionsService}
}

func (uc *ListAccountHealthUseCase) Execute(ctx context.Context) ([]AccountSyncStatus, error) {
	connections, err := uc.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]AccountSyncStatus, 0, len(connections))
	for _, conn := range connections {
		cursor, err := uc.repo.GetSyncCursor(ctx, conn.ID)
		if err != nil {
			return nil, err
		}

		var lastSyncedAt *string
		var status = "idle" // default

		if cursor != nil {
			status = cursor.Status
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
