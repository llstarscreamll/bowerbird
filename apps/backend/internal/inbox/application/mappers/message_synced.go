package mappers

import (
	contractEvents "github.com/bowerbird/internal/contracts/events"
	"github.com/bowerbird/internal/inbox/domain"
)

func ToInboxMessageReceived(event domain.MessageSynced) contractEvents.InboxMessageReceived {
	attachmentRefs := make([]contractEvents.AttachmentRef, 0, len(event.AttachmentRefs))
	for _, ref := range event.AttachmentRefs {
		attachmentRefs = append(attachmentRefs, contractEvents.AttachmentRef{
			S3Key:    ref.S3Key,
			Filename: ref.Filename,
			MimeType: ref.MimeType,
			SHA256:   ref.SHA256,
		})
	}

	return contractEvents.InboxMessageReceived{
		EventID:           event.EventID,
		OccurredAt:        event.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
		TenantID:          event.TenantSlug,
		AccountID:         event.AccountID,
		Provider:          event.Provider,
		ProviderMessageID: event.ProviderMessageID,
		MessageInternalID: event.MessageInternalID,
		Subject:           event.Subject,
		Body:              event.Body,
		Sender:            event.Sender,
		ReceivedAt:        event.ReceivedAt,
		AttachmentRefs:    attachmentRefs,
	}
}

func MarshalMessageSyncedPayload(event domain.MessageSynced) ([]byte, error) {
	return contractEvents.MarshalInboxMessageReceived(ToInboxMessageReceived(event))
}
