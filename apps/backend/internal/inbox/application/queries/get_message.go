package queries

import (
	"context"
	"encoding/json"

	"github.com/bowerbird/internal/inbox/application/ports"
	"github.com/bowerbird/internal/inbox/domain"
)

type MessageAttachmentSummary struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type MessageDetail struct {
	ID               string                     `json:"id"`
	Provider         string                     `json:"provider"`
	AccountID        string                     `json:"account_id"`
	AccountEmail     string                     `json:"account_email"`
	Subject          string                     `json:"subject"`
	Sender           string                     `json:"sender"`
	To               []string                   `json:"to,omitempty"`
	Cc               []string                   `json:"cc,omitempty"`
	Bcc              []string                   `json:"bcc,omitempty"`
	Snippet          string                     `json:"snippet"`
	ThreadID         string                     `json:"thread_id,omitempty"`
	Folder           string                     `json:"folder"`
	IsRead           bool                       `json:"is_read"`
	IsStarred        bool                       `json:"is_starred"`
	IsDraft          bool                       `json:"is_draft"`
	BodyText         string                     `json:"body_text"`
	BodyHTML         string                     `json:"body_html,omitempty"`
	ReceivedAt       string                     `json:"received_at"`
	ProcessingStatus string                     `json:"processing_status"`
	HasXML           bool                       `json:"has_xml"`
	HasPDF           bool                       `json:"has_pdf"`
	Attachments      []MessageAttachmentSummary `json:"attachments,omitempty"`
	ProviderMessage  *domain.MailMessage        `json:"provider_message,omitempty"`
}

type GetMessageQuery struct {
	repo ports.MessageQueryRepository
}

func NewGetMessageQuery(repo ports.MessageQueryRepository) *GetMessageQuery {
	if repo == nil {
		panic("message query repository is required")
	}
	return &GetMessageQuery{repo: repo}
}

func (q *GetMessageQuery) Execute(ctx context.Context, messageID, viewerUserID string) (*MessageDetail, error) {
	msg, err := q.repo.GetMessageViewByID(ctx, messageID, viewerUserID)
	if err != nil {
		return nil, err
	}

	var providerMessage *domain.MailMessage
	if len(msg.RawData) > 0 {
		var parsed domain.MailMessage
		if err := json.Unmarshal(msg.RawData, &parsed); err == nil {
			providerMessage = &parsed
		}
	}

	bodyText := msg.BodyText
	bodyHTML := ""
	if providerMessage != nil {
		if bodyText == "" {
			bodyText = providerMessage.PlainTextBody
		}
		bodyHTML = providerMessage.HTMLBody
	}

	attachments := make([]MessageAttachmentSummary, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, MessageAttachmentSummary{
			ID:        att.ID,
			Filename:  att.Filename,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}

	return &MessageDetail{
		ID:               msg.ID,
		Provider:         msg.Provider,
		AccountID:        msg.AccountID,
		AccountEmail:     msg.AccountEmail,
		Subject:          msg.Subject,
		Sender:           msg.Sender,
		To:               msg.ToEmails,
		Cc:               msg.CcEmails,
		Bcc:              msg.BccEmails,
		Snippet:          msg.Snippet,
		ThreadID:         msg.ThreadID,
		Folder:           msg.Folder,
		IsRead:           msg.IsRead,
		IsStarred:        msg.IsStarred,
		IsDraft:          msg.IsDraft,
		BodyText:         bodyText,
		BodyHTML:         bodyHTML,
		ReceivedAt:       msg.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingStatus: msg.ProcessingStatus,
		HasXML:           msg.HasXML,
		HasPDF:           msg.HasPDF,
		Attachments:      attachments,
		ProviderMessage:  providerMessage,
	}, nil
}
