package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type AccountHealthSummary struct {
	ID           string  `json:"id"`
	Provider     string  `json:"provider"`
	EmailAddress string  `json:"email_address"`
	Status       string  `json:"status"`
	LastSyncedAt *string `json:"last_synced_at,omitempty"`
}

type ListAccountHealthUseCase struct {
	repo domain.Repository
}

func NewListAccountHealthUseCase(repo domain.Repository) *ListAccountHealthUseCase {
	return &ListAccountHealthUseCase{repo: repo}
}

func (uc *ListAccountHealthUseCase) Execute(ctx context.Context) ([]AccountHealthSummary, error) {
	accounts, err := uc.repo.ListConnectedAccounts(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]AccountHealthSummary, 0, len(accounts))
	for _, acc := range accounts {
		var lastSyncedAt *string
		if acc.LastSyncedAt != nil {
			t := acc.LastSyncedAt.Format("2006-01-02T15:04:05Z07:00")
			lastSyncedAt = &t
		}

		summaries = append(summaries, AccountHealthSummary{
			ID:           acc.ID,
			Provider:     acc.Provider,
			EmailAddress: acc.EmailAddress,
			Status:       acc.Status,
			LastSyncedAt: lastSyncedAt,
		})
	}

	return summaries, nil
}
