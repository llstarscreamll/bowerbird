package application

import (
	"context"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type UnifiedMessageDetail struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	AccountID        string `json:"account_id"`
	AccountEmail     string `json:"account_email"`
	Subject          string `json:"subject"`
	Sender           string `json:"sender"`
	Snippet          string `json:"snippet"`
	BodyText         string `json:"body_text"`
	ReceivedAt       string `json:"received_at"`
	ProcessingStatus string `json:"processing_status"`
	HasXML           bool   `json:"has_xml"`
	HasPDF           bool   `json:"has_pdf"`
}

type GetMessageUseCase struct {
	repo domain.Repository
}

func NewGetMessageUseCase(repo domain.Repository) *GetMessageUseCase {
	return &GetMessageUseCase{repo: repo}
}

func (uc *GetMessageUseCase) Execute(ctx context.Context, messageID string) (*UnifiedMessageDetail, error) {
	msg, err := uc.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return &UnifiedMessageDetail{
		ID:               msg.ID,
		Provider:         msg.Provider,
		AccountID:        msg.AccountID,
		AccountEmail:     msg.AccountEmail,
		Subject:          msg.Subject,
		Sender:           msg.Sender,
		Snippet:          msg.Snippet,
		BodyText:         msg.BodyText,
		ReceivedAt:       msg.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProcessingStatus: msg.ProcessingStatus,
		HasXML:           msg.HasXML,
		HasPDF:           msg.HasPDF,
	}, nil
}
