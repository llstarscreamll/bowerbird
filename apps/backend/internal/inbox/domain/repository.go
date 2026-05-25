package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConnectedAccountNotFound = errors.New("connected account not found")
	ErrEmailMessageNotFound     = errors.New("email message not found")
)

type Repository interface {
	CreateConnectedAccount(ctx context.Context, account *ConnectedAccount) error
	GetConnectedAccountByID(ctx context.Context, accountID string) (*ConnectedAccount, error)
	ListConnectedAccountsByStatus(ctx context.Context, status string) ([]*ConnectedAccount, error)
	UpdateConnectedAccountSyncState(ctx context.Context, accountID, status string, lastSyncedAt *time.Time, lastError *string, updatedAt time.Time) error

	UpsertEmailMessage(ctx context.Context, message *EmailMessage) (bool, error)
	GetEmailMessageByProviderID(ctx context.Context, accountID, providerMessageID string) (*EmailMessage, error)

	UpsertEmailAttachment(ctx context.Context, attachment *EmailAttachment) (bool, error)
	ListEmailAttachmentsByMessageID(ctx context.Context, messageID string) ([]*EmailAttachment, error)
}
