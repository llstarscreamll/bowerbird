package domain

import (
	"testing"
	"time"
)

func TestInboxSyncCursorMarkSyncFailed(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	cursor := &InboxSyncCursor{ConnectionID: "conn-1", Status: InboxSyncStatusIdle}

	cursor.MarkSyncFailed(now, "provider timeout")

	if cursor.Status != InboxSyncStatusError {
		t.Fatalf("expected status error, got %s", cursor.Status)
	}
	if cursor.LastError == nil || *cursor.LastError != "provider timeout" {
		t.Fatalf("expected last error populated, got %#v", cursor.LastError)
	}
}

func TestInboxSyncCursorMarkSyncSucceeded(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	prevError := "failed"
	cursor := &InboxSyncCursor{ConnectionID: "conn-1", Status: InboxSyncStatusError, LastError: &prevError}

	cursor.MarkSyncSucceeded(now)

	if cursor.Status != InboxSyncStatusIdle {
		t.Fatalf("expected status idle, got %s", cursor.Status)
	}
	if cursor.LastError != nil {
		t.Fatalf("expected last error to be cleared")
	}
	if cursor.LastSyncedAt == nil || !cursor.LastSyncedAt.Equal(now) {
		t.Fatalf("expected last synced at %v, got %#v", now, cursor.LastSyncedAt)
	}
}

func TestNewSyncedEmailMessage(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	message, err := NewSyncedEmailMessage(NewEmailMessageInput{
		ID:                "msg-1",
		AccountID:         "acc-1",
		ProviderMessageID: "provider-msg-1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		t.Fatalf("new synced email message: %v", err)
	}
	if message.SyncStatus != EmailMessageSyncStatusSynced {
		t.Fatalf("expected sync status %q, got %q", EmailMessageSyncStatusSynced, message.SyncStatus)
	}
}

func TestNewEmailAttachmentRequiresFields(t *testing.T) {
	_, err := NewEmailAttachment(NewEmailAttachmentInput{})
	if err == nil {
		t.Fatal("expected required field error")
	}
}
