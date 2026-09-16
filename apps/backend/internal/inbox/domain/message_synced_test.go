package domain_test

import (
	"testing"
	"time"

	"github.com/bowerbird/internal/inbox/domain"
	"github.com/stretchr/testify/require"
)

func TestMessageSyncedOnlyOnFirstPersist(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	message, err := domain.NewInboxMessageFromProvider(domain.NewInboxMessageFromProviderInput{
		ID:           "msg-1",
		ConnectionID: "acc-1",
		ProviderMessage: &domain.MailMessage{
			ID:            "provider-msg-1",
			Subject:       "Invoice",
			Sender:        "billing@vendor.com",
			PlainTextBody: "body",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)

	ctx := domain.SyncNotificationContext{
		EventID:    "evt-1",
		TenantSlug: "tenant-a",
		AccountID:  "acc-1",
		Provider:   "gmail",
		ProviderMessage: &domain.MailMessage{
			ID:            "provider-msg-1",
			Subject:       "Invoice",
			Sender:        "billing@vendor.com",
			PlainTextBody: "body",
		},
	}

	existing, err := message.NotificationAfterPersist(false, ctx)
	require.NoError(t, err)
	require.Nil(t, existing)

	synced, err := message.NotificationAfterPersist(true, ctx)
	require.NoError(t, err)
	require.NotNil(t, synced)
	require.Equal(t, "msg-1", synced.MessageInternalID)
	require.Equal(t, "provider-msg-1", synced.ProviderMessageID)
	require.Equal(t, "Invoice", synced.Subject)
}

func TestNotificationAfterCaptureOnlyWhenFirstFullContent(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	stub, err := domain.NewInboxMessageAsSynced(domain.NewInboxMessageInput{
		ID:                "msg-1",
		ConnectionID:      "acc-1",
		ProviderMessageID: "provider-msg-1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	require.NoError(t, err)

	ctx := domain.SyncNotificationContext{
		EventID:    "evt-1",
		TenantSlug: "tenant-a",
		AccountID:  "acc-1",
		Provider:   "gmail",
		ProviderMessage: &domain.MailMessage{
			ID:            "provider-msg-1",
			Subject:       "Invoice",
			PlainTextBody: "body",
		},
	}

	none, err := stub.NotificationAfterCapture(false, ctx)
	require.NoError(t, err)
	require.Nil(t, none)

	require.NoError(t, stub.ApplyProviderMessage(ctx.ProviderMessage, []byte(`{"plain_text_body":"body"}`), now))
	first, err := stub.NotificationAfterCapture(false, ctx)
	require.NoError(t, err)
	require.NotNil(t, first)

	again, err := stub.NotificationAfterCapture(true, ctx)
	require.NoError(t, err)
	require.Nil(t, again)
}

func TestNotificationAfterCaptureSkipsNewsletter(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	stub, err := domain.NewInboxMessageAsSynced(domain.NewInboxMessageInput{
		ID:                "msg-1",
		ConnectionID:      "acc-1",
		ProviderMessageID: "provider-msg-1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	require.NoError(t, err)

	mail := &domain.MailMessage{
		ID:            "provider-msg-1",
		Subject:       "Reunión semanal",
		PlainTextBody: "agenda",
	}
	require.NoError(t, stub.ApplyProviderMessage(mail, []byte(`{"plain_text_body":"agenda"}`), now))

	event, err := stub.NotificationAfterCapture(false, domain.SyncNotificationContext{
		EventID:         "evt-1",
		TenantSlug:      "tenant-a",
		AccountID:       "acc-1",
		Provider:        "gmail",
		ProviderMessage: mail,
	})
	require.NoError(t, err)
	require.Nil(t, event)
}
