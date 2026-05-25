package domain

import (
	"context"
	"time"
)

const (
	ProviderGmail     = "gmail"
	ProviderOutlook   = "outlook"
	ProviderHotmail   = "hotmail"
	ProviderYahoo     = "yahoo"
	ProviderMicrosoft = "microsoft"
)

type ListMessagesOptions struct {
	UserID     string
	LabelIDs   []string
	Query      string
	PageToken  string
	MaxResults int
}

type MessageRef struct {
	ID       string
	ThreadID string
}

type MailAttachmentRef struct {
	AttachmentID string
	Filename     string
	MimeType     string
	Size         int64
}

type MailMessage struct {
	ID           string
	ThreadID     string
	Subject      string
	Sender       string
	ReceivedAt   *time.Time
	InternalDate *time.Time
	Attachments  []MailAttachmentRef
}

type DownloadedMailAttachment struct {
	MailAttachmentRef
	Data []byte
}

// MailProviderClient is the provider-agnostic inbox port implemented by each provider adapter.
type MailProviderClient interface {
	ListMessages(ctx context.Context, opts ListMessagesOptions) ([]MessageRef, string, error)
	GetMessage(ctx context.Context, userID, messageID string) (*MailMessage, error)
	DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error)
	DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []MailAttachmentRef) ([]DownloadedMailAttachment, error)
}
