package queries

import (
	"context"

	"github.com/bowerbird/internal/inbox/application/ports"
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

type ListMessagesQuery struct {
	repo ports.MessageQueryRepository
}

func NewListMessagesQuery(repo ports.MessageQueryRepository) *ListMessagesQuery {
	return &ListMessagesQuery{repo: repo}
}

func (q *ListMessagesQuery) Execute(ctx context.Context) ([]MessageSummary, error) {
	messages, err := q.repo.ListMessageViews(ctx)
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
