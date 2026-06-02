package domain

import (
	"testing"
	"time"
)

func TestInboxSyncCursorMarkSyncFailed(t *testing.T) {
	cursor := &SyncCursor{ConnectionID: "conn-1", Status: SyncCursorStatusIdle}

	cursor.MarkSyncFailed("provider timeout")

	if cursor.Status != SyncCursorStatusError {
		t.Fatalf("expected status error, got %s", cursor.Status)
	}
	if cursor.LastError == nil || *cursor.LastError != "provider timeout" {
		t.Fatalf("expected last error populated, got %#v", cursor.LastError)
	}
}

func TestInboxSyncCursorMarkSyncSucceeded(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	prevError := "failed"
	cursor := &SyncCursor{ConnectionID: "conn-1", Status: SyncCursorStatusError, LastError: &prevError}

	cursor.MarkSyncSucceeded(now)

	if cursor.Status != SyncCursorStatusIdle {
		t.Fatalf("expected status idle, got %s", cursor.Status)
	}
	if cursor.LastError != nil {
		t.Fatalf("expected last error to be cleared")
	}
	if cursor.LastSyncedAt == nil || !cursor.LastSyncedAt.Equal(now) {
		t.Fatalf("expected last synced at %v, got %#v", now, cursor.LastSyncedAt)
	}
}

func TestInboxSyncCursorMarkSyncing(t *testing.T) {
	cursor := &SyncCursor{ConnectionID: "conn-1", Status: SyncCursorStatusIdle}

	cursor.MarkSyncing()

	if cursor.Status != SyncCursorStatusSyncing {
		t.Fatalf("expected status syncing, got %s", cursor.Status)
	}
}

func TestNewInboxMessageAsSynced(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	message, err := NewInboxMessageAsSynced(NewInboxMessageInput{
		ID:                "msg-1",
		ConnectionID:      "acc-1",
		ProviderMessageID: "provider-msg-1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		t.Fatalf("new synced email message: %v", err)
	}
	if message.SyncStatus != MessageSyncStatusSynced {
		t.Fatalf("expected sync status %q, got %q", MessageSyncStatusSynced, message.SyncStatus)
	}
}

func TestNewEmailAttachmentRequiresFields(t *testing.T) {
	_, err := NewMessageAttachment(NewMessageAttachmentInput{})
	if err == nil {
		t.Fatal("expected required field error")
	}
}

func TestNewInboxMessageFromProvider(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	receivedAt := now.Add(-time.Hour)

	message, err := NewInboxMessageFromProvider(NewInboxMessageFromProviderInput{
		ID:           "msg-1",
		ConnectionID: "acc-1",
		ProviderMessage: &MailMessage{
			ID:         "provider-msg-1",
			ThreadID:   "thread-1",
			Subject:    "Invoice",
			Sender:     "sender@example.com",
			ReceivedAt: &receivedAt,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("new inbox message from provider: %v", err)
	}

	if message.ProviderMessageID != "provider-msg-1" {
		t.Fatalf("expected provider message id provider-msg-1, got %s", message.ProviderMessageID)
	}
	if message.ProviderThreadID == nil || *message.ProviderThreadID != "thread-1" {
		t.Fatalf("expected provider thread id thread-1, got %#v", message.ProviderThreadID)
	}
	if message.Subject == nil || *message.Subject != "Invoice" {
		t.Fatalf("expected subject Invoice, got %#v", message.Subject)
	}
	if message.SenderEmail == nil || *message.SenderEmail != "sender@example.com" {
		t.Fatalf("expected sender sender@example.com, got %#v", message.SenderEmail)
	}
	if message.ReceivedAt == nil || !message.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("expected received at %v, got %#v", receivedAt, message.ReceivedAt)
	}
}
