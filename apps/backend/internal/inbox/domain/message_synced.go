package domain

import "time"

// MessageSynced is a domain event raised when a new inbox message is first persisted.
type MessageSynced struct {
	EventID           string
	OccurredAt        time.Time
	TenantSlug        string
	AccountID         string
	Provider          string
	ProviderMessageID string
	MessageInternalID string
	Subject           string
	Body              string
	Sender            string
	ReceivedAt        string
	AttachmentRefs    []SyncedAttachmentRef
}

type SyncedAttachmentRef struct {
	S3Key    string
	Filename string
	MimeType string
	SHA256   string
}

// SyncNotificationContext carries cross-cutting data required to build MessageSynced.
type SyncNotificationContext struct {
	EventID         string
	TenantSlug      string
	AccountID       string
	Provider        string
	ProviderMessage *MailMessage
	AttachmentRefs  []SyncedAttachmentRef
}

// NotificationAfterPersist returns a domain event only when the message was newly inserted.
func (m *InboxMessage) NotificationAfterPersist(inserted bool, ctx SyncNotificationContext) (*MessageSynced, error) {
	if !inserted {
		return nil, nil
	}
	if ctx.ProviderMessage == nil {
		return nil, ErrInboxMessageProviderIDRequired
	}
	if ctx.EventID == "" {
		return nil, ErrInboxMessageIDRequired
	}
	if ctx.TenantSlug == "" || ctx.AccountID == "" || ctx.Provider == "" {
		return nil, ErrInboxMessageConnectionIDRequired
	}

	event := &MessageSynced{
		EventID:           ctx.EventID,
		OccurredAt:        m.createdAt,
		TenantSlug:        ctx.TenantSlug,
		AccountID:         ctx.AccountID,
		Provider:          ctx.Provider,
		ProviderMessageID: ctx.ProviderMessage.ID,
		MessageInternalID: m.id,
		AttachmentRefs:    append([]SyncedAttachmentRef(nil), ctx.AttachmentRefs...),
	}

	if ctx.ProviderMessage.Subject != "" {
		event.Subject = ctx.ProviderMessage.Subject
	}
	if ctx.ProviderMessage.PlainTextBody != "" {
		event.Body = ctx.ProviderMessage.PlainTextBody
	}
	if ctx.ProviderMessage.Sender != "" {
		event.Sender = ctx.ProviderMessage.Sender
	}
	if ctx.ProviderMessage.ReceivedAt != nil {
		event.ReceivedAt = ctx.ProviderMessage.ReceivedAt.Format(time.RFC3339)
	}

	return event, nil
}
