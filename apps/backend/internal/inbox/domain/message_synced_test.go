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
