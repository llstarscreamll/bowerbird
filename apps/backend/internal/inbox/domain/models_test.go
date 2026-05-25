package domain

import (
	"testing"
	"time"
)

func TestConnectedAccountMarkSyncFailed(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	account := &ConnectedAccount{ID: "acc-1", Status: ConnectedAccountStatusActive}

	if err := account.MarkSyncFailed(now, "provider timeout"); err != nil {
		t.Fatalf("mark sync failed: %v", err)
	}

	if account.Status != ConnectedAccountStatusError {
		t.Fatalf("expected status error, got %s", account.Status)
	}
	if account.LastError == nil || *account.LastError != "provider timeout" {
		t.Fatalf("expected last error populated, got %#v", account.LastError)
	}

	events := account.PullSyncStateEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(events))
	}
	if events[0].FromStatus != ConnectedAccountStatusActive || events[0].ToStatus != ConnectedAccountStatusError {
		t.Fatalf("unexpected transition event: %#v", events[0])
	}
}

func TestConnectedAccountMarkSyncSucceeded(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	prevError := "failed"
	account := &ConnectedAccount{ID: "acc-1", Status: ConnectedAccountStatusError, LastError: &prevError}

	if err := account.MarkSyncSucceeded(now); err != nil {
		t.Fatalf("mark sync succeeded: %v", err)
	}

	if account.Status != ConnectedAccountStatusActive {
		t.Fatalf("expected status active, got %s", account.Status)
	}
	if account.LastError != nil {
		t.Fatalf("expected last error to be cleared")
	}
	if account.LastSyncedAt == nil || !account.LastSyncedAt.Equal(now) {
		t.Fatalf("expected last synced at %v, got %#v", now, account.LastSyncedAt)
	}
}

func TestConnectedAccountMarkSyncFailedRequiresReason(t *testing.T) {
	account := &ConnectedAccount{ID: "acc-1", Status: ConnectedAccountStatusActive}

	err := account.MarkSyncFailed(time.Now(), "")
	if err == nil {
		t.Fatal("expected error for empty failure reason")
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
