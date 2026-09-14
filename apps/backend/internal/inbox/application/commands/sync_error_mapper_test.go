package commands

import (
	"errors"
	"testing"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

func TestClassifySyncError_Reauth(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	err := classifySyncError(account, errors.New("provider request failed with status 401"))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncReauthRequired {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncReauthRequired, syncErr.Code)
	}
	if !syncErr.RequiresReauth {
		t.Fatalf("expected requires_reauth=true")
	}
}

func TestClassifySyncError_ForbiddenAttachmentIsNotReauth(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	err := classifySyncError(account, errors.New("download attachment request failed with status 403"))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncInternal {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncInternal, syncErr.Code)
	}
	if syncErr.RequiresReauth {
		t.Fatalf("expected requires_reauth=false")
	}
	if shouldMarkRequiresReconnect(err) {
		t.Fatal("attachment 403 must not mark requires_reconnect")
	}
}

func TestClassifySyncError_InsufficientScopesIsReauth(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	err := classifySyncError(account, errors.New("request had insufficient authentication scopes"))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncReauthRequired {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncReauthRequired, syncErr.Code)
	}
}

func TestClassifySyncError_GmailQuotaExceededIsRateLimited(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	err := classifySyncError(account, errors.New(`modify message request failed with status 403 (body="Quota exceeded for quota metric 'Total Query Cost' reason: rateLimitExceeded")`))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncRateLimited {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncRateLimited, syncErr.Code)
	}
	if syncErr.RetryAfterSeconds != 120 {
		t.Fatalf("expected retry_after_seconds=120, got %d", syncErr.RetryAfterSeconds)
	}
	if isSkippableAttachmentError(err) {
		t.Fatal("quota 403 must not be skipped as a forbidden attachment")
	}
}

func TestClassifySyncError_RateLimitedRetryAfter(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "yahoo", ProviderAccountEmail: "user@yahoo.com"}
	err := classifySyncError(account, errors.New("request failed with status 429 (retry-after=\"180\")"))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncRateLimited {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncRateLimited, syncErr.Code)
	}
	if syncErr.RetryAfterSeconds != 180 {
		t.Fatalf("expected retry_after_seconds=180, got %d", syncErr.RetryAfterSeconds)
	}
}

func TestClassifySyncError_PayloadRejected(t *testing.T) {
	account := connectionsapi.ConnectionInfo{Provider: "outlook", ProviderAccountEmail: "user@outlook.com"}
	err := classifySyncError(account, errors.Join(errors.New("payload too large"), errPayloadRejected))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncPayloadRejected {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncPayloadRejected, syncErr.Code)
	}
}
