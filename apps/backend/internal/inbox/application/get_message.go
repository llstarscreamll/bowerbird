package application

import (
	"context"
	"encoding/json"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type MessageDetail struct {
	ID               string              `json:"id"`
	Provider         string              `json:"provider"`
	AccountID        string              `json:"account_id"`
	AccountEmail     string              `json:"account_email"`
	Subject          string              `json:"subject"`
	Sender           string              `json:"sender"`
	Snippet          string              `json:"snippet"`
	BodyText         string              `json:"body_text"`
	BodyHTML         string              `json:"body_html,omitempty"`
	ReceivedAt       string              `json:"received_at"`
	ProcessingStatus string              `json:"processing_status"`
	HasXML           bool                `json:"has_xml"`
	HasPDF           bool                `json:"has_pdf"`
	ProviderMessage  *domain.MailMessage `json:"provider_message,omitempty"`
}

type GetMessageUseCase struct {
	repo MessageQueryRepository
}

func NewGetMessageUseCase(repo MessageQueryRepository) *GetMessageUseCase {
	return &GetMessageUseCase{repo: repo}
}

func (uc *GetMessageUseCase) Execute(ctx context.Context, messageID string) (*MessageDetail, error) {
	msg, err := uc.repo.GetMessageViewByID(ctx, messageID)
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

	return &MessageDetail{
		ID:               msg.ID,
		Provider:         msg.Provider,
		AccountID:        msg.AccountID,
		AccountEmail:     msg.AccountEmail,
		Subject:          msg.Subject,
		Sender:           msg.Sender,
		Snippet:          msg.Snippet,
		BodyText:         bodyText,
		BodyHTML:         bodyHTML,
		ReceivedAt:       msg.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingStatus: msg.ProcessingStatus,
		HasXML:           msg.HasXML,
		HasPDF:           msg.HasPDF,
		ProviderMessage:  providerMessage,
	}, nil
}
