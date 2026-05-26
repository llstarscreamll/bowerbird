package domain

import "time"

const (
	ConnectedAccountStatusActive            = "active"
	ConnectedAccountStatusRequiresReconnect = "requires_reconnect"
	ConnectedAccountStatusPaused            = "paused"
	ConnectedAccountStatusError             = "error"
)

type ConnectedAccountSyncStateChanged struct {
	AccountID    string
	FromStatus   string
	ToStatus     string
	OccurredAt   time.Time
	LastSyncedAt *time.Time
	LastError    *string
}

type ConnectedAccount struct {
	ID           string
	Provider     string
	EmailAddress string
	Status       string
	// EncryptedCredentials stores provider tokens encrypted at application layer.
	EncryptedCredentials []byte
	LastSyncedAt         *time.Time
	LastError            *string
	RawData              []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
	pendingEvents        []ConnectedAccountSyncStateChanged
}

func (a *ConnectedAccount) MarkSyncFailed(at time.Time, failure string) error {
	if a == nil {
		return ErrNilConnectedAccount
	}
	if failure == "" {
		return ErrSyncFailureReasonRequired
	}

	next := ConnectedAccountStatusError
	lastError := failure
	a.transitionSyncState(next, at.UTC(), a.LastSyncedAt, &lastError)

	return nil
}

func (a *ConnectedAccount) MarkSyncSucceeded(at time.Time) error {
	if a == nil {
		return ErrNilConnectedAccount
	}

	syncedAt := at.UTC()
	next := ConnectedAccountStatusActive
	a.transitionSyncState(next, syncedAt, &syncedAt, nil)

	return nil
}

func (a *ConnectedAccount) PullSyncStateEvents() []ConnectedAccountSyncStateChanged {
	if len(a.pendingEvents) == 0 {
		return nil
	}

	events := make([]ConnectedAccountSyncStateChanged, len(a.pendingEvents))
	copy(events, a.pendingEvents)
	a.pendingEvents = nil

	return events
}

func (a *ConnectedAccount) transitionSyncState(nextStatus string, at time.Time, lastSyncedAt *time.Time, lastError *string) {
	previous := a.Status
	a.Status = nextStatus
	a.LastSyncedAt = lastSyncedAt
	a.LastError = lastError
	a.UpdatedAt = at

	a.pendingEvents = append(a.pendingEvents, ConnectedAccountSyncStateChanged{
		AccountID:    a.ID,
		FromStatus:   previous,
		ToStatus:     nextStatus,
		OccurredAt:   at,
		LastSyncedAt: lastSyncedAt,
		LastError:    lastError,
	})
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
	ReceivedAt       time.Time
	ProcessingStatus string
	HasXML           bool
	HasPDF           bool
}
