package domain

import "time"

const (
	InboxSyncStatusSyncing = "syncing"
	InboxSyncStatusIdle    = "idle"
	InboxSyncStatusError   = "error"
)

type InboxSyncCursor struct {
	ConnectionID string
	LastSyncedAt *time.Time
	LastError    *string
	Status       string
}

func (c *InboxSyncCursor) MarkSyncFailed(at time.Time, failure string) {
	c.Status = InboxSyncStatusError
	c.LastError = &failure
}

func (c *InboxSyncCursor) MarkSyncSucceeded(at time.Time) {
	c.Status = InboxSyncStatusIdle
	c.LastError = nil
	syncedAt := at.UTC()
	c.LastSyncedAt = &syncedAt
}

type EmailMessage struct {
	ID                string
	AccountID         string
	ProviderMessageID string
	ProviderThreadID  *string
	Subject           *string
	SenderEmail       *string
	ReceivedAt        *time.Time
	SyncStatus        string
	RawData           []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type EmailAttachment struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  *string
	SizeBytes *int64
	SHA256    string
	S3Key     string
	RawData   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

const EmailMessageSyncStatusSynced = "synced"

type NewEmailMessageInput struct {
	ID                string
	AccountID         string
	ProviderMessageID string
	ProviderThreadID  *string
	Subject           *string
	SenderEmail       *string
	ReceivedAt        *time.Time
	RawData           []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewSyncedEmailMessage(input NewEmailMessageInput) (*EmailMessage, error) {
	if input.ID == "" {
		return nil, ErrEmailMessageIDRequired
	}
	if input.AccountID == "" {
		return nil, ErrEmailMessageAccountIDRequired
	}
	if input.ProviderMessageID == "" {
		return nil, ErrEmailMessageProviderIDRequired
	}

	return &EmailMessage{
		ID:                input.ID,
		AccountID:         input.AccountID,
		ProviderMessageID: input.ProviderMessageID,
		ProviderThreadID:  input.ProviderThreadID,
		Subject:           input.Subject,
		SenderEmail:       input.SenderEmail,
		ReceivedAt:        input.ReceivedAt,
		SyncStatus:        EmailMessageSyncStatusSynced,
		RawData:           input.RawData,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	}, nil
}

type NewEmailAttachmentInput struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  *string
	SizeBytes *int64
	SHA256    string
	S3Key     string
	RawData   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewEmailAttachment(input NewEmailAttachmentInput) (*EmailAttachment, error) {
	if input.ID == "" {
		return nil, ErrEmailAttachmentIDRequired
	}
	if input.MessageID == "" {
		return nil, ErrEmailAttachmentMessageIDRequired
	}
	if input.Filename == "" {
		return nil, ErrEmailAttachmentFilenameRequired
	}
	if input.SHA256 == "" {
		return nil, ErrEmailAttachmentSHARequired
	}
	if input.S3Key == "" {
		return nil, ErrEmailAttachmentS3KeyRequired
	}

	return &EmailAttachment{
		ID:        input.ID,
		MessageID: input.MessageID,
		Filename:  input.Filename,
		MimeType:  input.MimeType,
		SizeBytes: input.SizeBytes,
		SHA256:    input.SHA256,
		S3Key:     input.S3Key,
		RawData:   input.RawData,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
	}, nil
}

type UnifiedMessage struct {
	ID               string
	Provider         string
	AccountID        string
	AccountEmail     string
	Subject          string
	Sender           string
	Snippet          string
	BodyText         string
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
}
