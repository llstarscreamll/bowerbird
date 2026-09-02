package mappers_test

import (
	"testing"
	"time"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	inboxMappers "github.com/bowerbird/internal/inbox/application/mappers"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/stretchr/testify/require"
)

func TestMarshalMessageSyncedPayload(t *testing.T) {
	occurredAt := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	body, err := inboxMappers.MarshalMessageSyncedPayload(domain.MessageSynced{
		EventID:           "evt-1",
		OccurredAt:        occurredAt,
		TenantSlug:        "tenant-a",
		AccountID:         "acc-1",
		Provider:          "gmail",
		ProviderMessageID: "provider-msg-1",
		MessageInternalID: "msg-1",
		Subject:           "Invoice",
		Body:              "body",
		Sender:            "billing@vendor.com",
		AttachmentRefs: []domain.SyncedAttachmentRef{{
			S3Key: "tenant/a/file.xml", Filename: "file.xml", MimeType: "application/xml",
		}},
	})
	require.NoError(t, err)

	decoded, err := contractEvents.UnmarshalInboxMessageReceived(body)
	require.NoError(t, err)
	require.Equal(t, "evt-1", decoded.EventID)
	require.Equal(t, "tenant-a", decoded.TenantID)
	require.Equal(t, "msg-1", decoded.MessageInternalID)
	require.Len(t, decoded.AttachmentRefs, 1)
}

func TestToInboxMessageReceivedMapsFields(t *testing.T) {
	mapped := inboxMappers.ToInboxMessageReceived(domain.MessageSynced{
		EventID:           "evt-1",
		OccurredAt:        time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
		TenantSlug:        "tenant-a",
		AccountID:         "acc-1",
		Provider:          "gmail",
		ProviderMessageID: "provider-msg-1",
		MessageInternalID: "msg-1",
		Subject:           "Invoice",
		Body:              "body",
		Sender:            "billing@vendor.com",
		ReceivedAt:        "2026-05-25T12:00:00Z",
	})
	require.Equal(t, contractEvents.InboxMessageReceivedDetailType, "InboxMessageReceived")
	require.Equal(t, "Invoice", mapped.Subject)
	require.Equal(t, "billing@vendor.com", mapped.Sender)
}
