package domain

import (
	"context"
	"errors"
)

var (
	ErrConnectedAccountNotFound = errors.New("connected account not found")
	ErrEmailMessageNotFound     = errors.New("email message not found")
)

type Repository interface {
	GetSyncCursor(ctx context.Context, connectionID string) (*InboxSyncCursor, error)
	UpsertSyncCursor(ctx context.Context, cursor *InboxSyncCursor) error

	UpsertEmailMessage(ctx context.Context, msg *EmailMessage) (bool, error)
	UpsertEmailAttachment(ctx context.Context, attachment *EmailAttachment) (bool, error)

	ListUnifiedMessages(ctx context.Context) ([]UnifiedMessage, error)
	ListMessagesByAccount(ctx context.Context, accountID string, limit, offset int) ([]UnifiedMessage, error)
	GetMessageByID(ctx context.Context, messageID string) (*UnifiedMessage, error)
	GetMessageAttachments(ctx context.Context, messageID string) ([]EmailAttachment, error)
}
