package application

import (
	"context"
)

type MessageSummary struct {
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
	repo MessageQueryRepository
}

func NewListMessagesUseCase(repo MessageQueryRepository) *ListMessagesUseCase {
	return &ListMessagesUseCase{repo: repo}
}

func (uc *ListMessagesUseCase) Execute(ctx context.Context) ([]MessageSummary, error) {
	messages, err := uc.repo.ListMessageViews(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]MessageSummary, 0, len(messages))
	for _, msg := range messages {
		summaries = append(summaries, MessageSummary{
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
