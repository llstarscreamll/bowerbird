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
	ToEmails          []string
	CcEmails          []string
	BccEmails         []string
	Snippet           *string
	Folder            MailFolder
	IsRead            bool
	IsStarred         bool
	IsDraft           bool
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
	ToEmails          []string
	CcEmails          []string
	BccEmails         []string
	Snippet           *string
	Folder            MailFolder
	IsRead            bool
	IsStarred         bool
	IsDraft           bool
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

	folder := input.Folder
	if folder == "" {
		folder = MailFolderInbox
	}

	return &InboxMessage{
		ID:                input.ID,
		ConnectionID:      input.ConnectionID,
		ProviderMessageID: input.ProviderMessageID,
		ProviderThreadID:  input.ProviderThreadID,
		Subject:           input.Subject,
		SenderEmail:       input.SenderEmail,
		ToEmails:          input.ToEmails,
		CcEmails:          input.CcEmails,
		BccEmails:         input.BccEmails,
		Snippet:           input.Snippet,
		Folder:            folder,
		IsRead:            input.IsRead,
		IsStarred:         input.IsStarred,
		IsDraft:           input.IsDraft,
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

	flags := FlagsFromProviderLabels(input.ProviderMessage.LabelIDs)

	return NewInboxMessageAsSynced(NewInboxMessageInput{
		ID:                input.ID,
		ConnectionID:      input.ConnectionID,
		ProviderMessageID: input.ProviderMessage.ID,
		ProviderThreadID:  optionalStringPointer(input.ProviderMessage.ThreadID),
		Subject:           optionalStringPointer(input.ProviderMessage.Subject),
		SenderEmail:       optionalStringPointer(input.ProviderMessage.Sender),
		ToEmails:          firstNonEmptyStrings(input.ProviderMessage.To, ParseAddressList(headerValueFromMail(input.ProviderMessage, "to"))),
		CcEmails:          firstNonEmptyStrings(input.ProviderMessage.Cc, ParseAddressList(headerValueFromMail(input.ProviderMessage, "cc"))),
		BccEmails:         firstNonEmptyStrings(input.ProviderMessage.Bcc, ParseAddressList(headerValueFromMail(input.ProviderMessage, "bcc"))),
		Snippet:           optionalStringPointer(input.ProviderMessage.Snippet),
		Folder:            flags.Folder,
		IsRead:            flags.IsRead,
		IsStarred:         flags.IsStarred,
		IsDraft:           flags.IsDraft,
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

func headerValueFromMail(message *MailMessage, name string) string {
	if message == nil {
		return ""
	}
	for _, header := range message.Headers {
		if equalFoldASCII(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func firstNonEmptyStrings(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ac, bc := a[i], b[i]
		if ac >= 'A' && ac <= 'Z' {
			ac += 'a' - 'A'
		}
		if bc >= 'A' && bc <= 'Z' {
			bc += 'a' - 'A'
		}
		if ac != bc {
			return false
		}
	}
	return true
}
