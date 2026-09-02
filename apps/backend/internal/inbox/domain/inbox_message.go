package domain

import "time"

type MessageSyncStatus string

const MessageSyncStatusSynced MessageSyncStatus = "synced"

// InboxMessage is the inbox message aggregate root.
type InboxMessage struct {
	id                string
	connectionID      string
	providerMessageID string
	providerThreadID  *string
	subject           *string
	senderEmail       *string
	toEmails          []string
	ccEmails          []string
	bccEmails         []string
	snippet           *string
	folder            MailFolder
	isRead            bool
	isStarred         bool
	isDraft           bool
	receivedAt        *time.Time
	syncStatus        MessageSyncStatus
	rawData           []byte
	createdAt         time.Time
	updatedAt         time.Time
}

// InboxMessageSnapshot is the persistence representation of InboxMessage.
type InboxMessageSnapshot struct {
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

func (m *InboxMessage) ID() string                    { return m.id }
func (m *InboxMessage) ConnectionID() string          { return m.connectionID }
func (m *InboxMessage) ProviderMessageID() string     { return m.providerMessageID }
func (m *InboxMessage) ProviderThreadID() *string     { return m.providerThreadID }
func (m *InboxMessage) Subject() *string              { return m.subject }
func (m *InboxMessage) SenderEmail() *string          { return m.senderEmail }
func (m *InboxMessage) Folder() MailFolder            { return m.folder }
func (m *InboxMessage) IsRead() bool                  { return m.isRead }
func (m *InboxMessage) IsStarred() bool               { return m.isStarred }
func (m *InboxMessage) IsDraft() bool                 { return m.isDraft }
func (m *InboxMessage) CreatedAt() time.Time          { return m.createdAt }
func (m *InboxMessage) UpdatedAt() time.Time          { return m.updatedAt }
func (m *InboxMessage) SyncStatus() MessageSyncStatus { return m.syncStatus }
func (m *InboxMessage) ReceivedAt() *time.Time        { return m.receivedAt }

func (m *InboxMessage) Snapshot() InboxMessageSnapshot {
	return InboxMessageSnapshot{
		ID:                m.id,
		ConnectionID:      m.connectionID,
		ProviderMessageID: m.providerMessageID,
		ProviderThreadID:  m.providerThreadID,
		Subject:           m.subject,
		SenderEmail:       m.senderEmail,
		ToEmails:          append([]string(nil), m.toEmails...),
		CcEmails:          append([]string(nil), m.ccEmails...),
		BccEmails:         append([]string(nil), m.bccEmails...),
		Snippet:           m.snippet,
		Folder:            m.folder,
		IsRead:            m.isRead,
		IsStarred:         m.isStarred,
		IsDraft:           m.isDraft,
		ReceivedAt:        m.receivedAt,
		SyncStatus:        m.syncStatus,
		RawData:           append([]byte(nil), m.rawData...),
		CreatedAt:         m.createdAt,
		UpdatedAt:         m.updatedAt,
	}
}

func RehydrateInboxMessage(snapshot InboxMessageSnapshot) *InboxMessage {
	return &InboxMessage{
		id:                snapshot.ID,
		connectionID:      snapshot.ConnectionID,
		providerMessageID: snapshot.ProviderMessageID,
		providerThreadID:  snapshot.ProviderThreadID,
		subject:           snapshot.Subject,
		senderEmail:       snapshot.SenderEmail,
		toEmails:          append([]string(nil), snapshot.ToEmails...),
		ccEmails:          append([]string(nil), snapshot.CcEmails...),
		bccEmails:         append([]string(nil), snapshot.BccEmails...),
		snippet:           snapshot.Snippet,
		folder:            snapshot.Folder,
		isRead:            snapshot.IsRead,
		isStarred:         snapshot.IsStarred,
		isDraft:           snapshot.IsDraft,
		receivedAt:        snapshot.ReceivedAt,
		syncStatus:        snapshot.SyncStatus,
		rawData:           append([]byte(nil), snapshot.RawData...),
		createdAt:         snapshot.CreatedAt,
		updatedAt:         snapshot.UpdatedAt,
	}
}

func (m *InboxMessage) ConfirmPersisted(id string) {
	if id != "" {
		m.id = id
	}
}

func (m *InboxMessage) MarkAsRead(now time.Time) {
	m.isRead = true
	m.updatedAt = now.UTC()
}

func (m *InboxMessage) MarkAsUnread(now time.Time) {
	m.isRead = false
	m.updatedAt = now.UTC()
}

func (m *InboxMessage) Star(now time.Time) {
	m.isStarred = true
	m.updatedAt = now.UTC()
}

func (m *InboxMessage) Unstar(now time.Time) {
	m.isStarred = false
	m.updatedAt = now.UTC()
}

func (m *InboxMessage) Archive(now time.Time) {
	if m.folder == MailFolderInbox {
		m.folder = MailFolderArchive
	}
	m.updatedAt = now.UTC()
}

func (m *InboxMessage) MoveToTrash(now time.Time) {
	m.folder = MailFolderTrash
	m.updatedAt = now.UTC()
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

	return RehydrateInboxMessage(InboxMessageSnapshot{
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
	}), nil
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
