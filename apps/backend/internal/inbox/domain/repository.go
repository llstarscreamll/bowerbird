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
}
