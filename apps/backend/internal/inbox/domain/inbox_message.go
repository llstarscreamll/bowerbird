package domain

import "time"

type MessageSyncStatus string

const MessageSyncStatusSynced MessageSyncStatus = "synced"

type InboxMessage struct {
	ID                string
	ConnectionID      string
	ProviderMessageID string
	ProviderThreadID  *string
	Subject           *string
	SenderEmail       *string
	ReceivedAt        *time.Time
	SyncStatus        MessageSyncStatus
	RawData           []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type NewInboxMessageInput struct {
	ID                string
	ConnectionID      string
	ProviderMessageID string
	ProviderThreadID  *string
	Subject           *string
	SenderEmail       *string
	ReceivedAt        *time.Time
	RawData           []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type NewInboxMessageFromProviderInput struct {
	ID              string
	ConnectionID    string
	ProviderMessage *MailMessage
	RawData         []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewInboxMessageAsSynced(input NewInboxMessageInput) (*InboxMessage, error) {
	if input.ID == "" {
		return nil, ErrInboxMessageIDRequired
	}
	if input.ConnectionID == "" {
		return nil, ErrInboxMessageConnectionIDRequired
	}
	if input.ProviderMessageID == "" {
		return nil, ErrInboxMessageProviderIDRequired
	}

	return &InboxMessage{
		ID:                input.ID,
		ConnectionID:      input.ConnectionID,
		ProviderMessageID: input.ProviderMessageID,
		ProviderThreadID:  input.ProviderThreadID,
		Subject:           input.Subject,
		SenderEmail:       input.SenderEmail,
		ReceivedAt:        input.ReceivedAt,
		SyncStatus:        MessageSyncStatusSynced,
		RawData:           input.RawData,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	}, nil
}

func NewInboxMessageFromProvider(input NewInboxMessageFromProviderInput) (*InboxMessage, error) {
	if input.ProviderMessage == nil {
		return nil, ErrInboxMessageProviderIDRequired
	}

	return NewInboxMessageAsSynced(NewInboxMessageInput{
		ID:                input.ID,
		ConnectionID:      input.ConnectionID,
		ProviderMessageID: input.ProviderMessage.ID,
		ProviderThreadID:  optionalStringPointer(input.ProviderMessage.ThreadID),
		Subject:           optionalStringPointer(input.ProviderMessage.Subject),
		SenderEmail:       optionalStringPointer(input.ProviderMessage.Sender),
		ReceivedAt:        input.ProviderMessage.ReceivedAt,
		RawData:           input.RawData,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	})
}

func optionalStringPointer(value string) *string {
	if value == "" {
		return nil
	}

	v := value
	return &v
}
