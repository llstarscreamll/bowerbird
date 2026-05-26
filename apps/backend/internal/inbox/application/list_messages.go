package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type UnifiedMessageSummary struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	AccountID        string `json:"account_id"`
	AccountEmail     string `json:"account_email"`
	Subject          string `json:"subject"`
	Sender           string `json:"sender"`
	Snippet          string `json:"snippet"`
	ReceivedAt       string `json:"received_at"`
	ProcessingStatus string `json:"processing_status"`
	HasXML           bool   `json:"has_xml"`
	HasPDF           bool   `json:"has_pdf"`
}

type ListMessagesUseCase struct {
	repo domain.Repository
}

func NewListMessagesUseCase(repo domain.Repository) *ListMessagesUseCase {
	return &ListMessagesUseCase{repo: repo}
}

func (uc *ListMessagesUseCase) Execute(ctx context.Context) ([]UnifiedMessageSummary, error) {
	messages, err := uc.repo.ListUnifiedMessages(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]UnifiedMessageSummary, 0, len(messages))
	for _, msg := range messages {
		summaries = append(summaries, UnifiedMessageSummary{
			ID:               msg.ID,
			Provider:         msg.Provider,
			AccountID:        msg.AccountID,
			AccountEmail:     msg.AccountEmail,
			Subject:          msg.Subject,
			Sender:           msg.Sender,
			Snippet:          msg.Snippet,
			ReceivedAt:       msg.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
			ProcessingStatus: msg.ProcessingStatus,
			HasXML:           msg.HasXML,
			HasPDF:           msg.HasPDF,
		})
	}

	return summaries, nil
}
