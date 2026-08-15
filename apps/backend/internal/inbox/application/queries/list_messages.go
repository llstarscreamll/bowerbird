package queries

import (
	"context"

	"github.com/bowerbird/internal/inbox/application/ports"
)

type MessageSummary struct {
	ID               string   `json:"id"`
	Provider         string   `json:"provider"`
	AccountID        string   `json:"account_id"`
	AccountEmail     string   `json:"account_email"`
	Subject          string   `json:"subject"`
	Sender           string   `json:"sender"`
	Snippet          string   `json:"snippet"`
	To               []string `json:"to,omitempty"`
	ThreadID         string   `json:"thread_id,omitempty"`
	Folder           string   `json:"folder"`
	IsRead           bool     `json:"is_read"`
	IsStarred        bool     `json:"is_starred"`
	IsDraft          bool     `json:"is_draft"`
	ReceivedAt       string   `json:"received_at"`
	ProcessingStatus string   `json:"processing_status"`
	HasXML           bool     `json:"has_xml"`
	HasPDF           bool     `json:"has_pdf"`
}

type MessageListResult struct {
	Data   []MessageSummary `json:"data"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type ListMessagesQuery struct {
	repo ports.MessageQueryRepository
}

func NewListMessagesQuery(repo ports.MessageQueryRepository) *ListMessagesQuery {
	return &ListMessagesQuery{repo: repo}
}

func (q *ListMessagesQuery) Execute(ctx context.Context, filter ports.ListMessagesFilter) (MessageListResult, error) {
	page, err := q.repo.ListMessageViews(ctx, filter)
	if err != nil {
		return MessageListResult{}, err
	}

	summaries := make([]MessageSummary, 0, len(page.Items))
	for _, msg := range page.Items {
		summaries = append(summaries, MessageSummary{
			ID:               msg.ID,
			Provider:         msg.Provider,
			AccountID:        msg.AccountID,
			AccountEmail:     msg.AccountEmail,
			Subject:          msg.Subject,
			Sender:           msg.Sender,
			Snippet:          msg.Snippet,
			To:               msg.ToEmails,
			ThreadID:         msg.ThreadID,
			Folder:           msg.Folder,
			IsRead:           msg.IsRead,
			IsStarred:        msg.IsStarred,
			IsDraft:          msg.IsDraft,
			ReceivedAt:       msg.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
			ProcessingStatus: msg.ProcessingStatus,
			HasXML:           msg.HasXML,
			HasPDF:           msg.HasPDF,
		})
	}

	return MessageListResult{
		Data:   summaries,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}
