package domain

import (
	"context"
	"errors"
)

var (
	ErrInboxMessageNotFound = errors.New("inbox message not found")
)

type Repository interface {
	SyncCursorRepository
	MessageRepository
}

type SyncCursorRepository interface {
	GetSyncCursor(ctx context.Context, connectionID string) (*SyncCursor, error)
	UpsertSyncCursor(ctx context.Context, cursor *SyncCursor) error
}

type MessageRepository interface {
	UpsertInboxMessage(ctx context.Context, msg *InboxMessage) (bool, error)
	UpsertMessageAttachment(ctx context.Context, attachment *MessageAttachment) (bool, error)
	GetInboxMessageByID(ctx context.Context, messageID string) (*InboxMessage, error)
	UpdateInboxMessageFlags(ctx context.Context, message *InboxMessage) error
	GetMessageAttachment(ctx context.Context, messageID, attachmentID string) (*MessageAttachment, error)
	ListMessageAttachments(ctx context.Context, messageID string) ([]*MessageAttachment, error)
}
