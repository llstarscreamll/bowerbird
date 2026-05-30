package application

import (
	"errors"
	"testing"

	connectionsapp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	appErrors "github.com/money-path/bowerbird/apps/backend/internal/platform/errors"
)

func TestClassifySyncError_Reauth(t *testing.T) {
	account := connectionsapp.ConnectionInfo{Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
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

func TestClassifySyncError_RateLimitedRetryAfter(t *testing.T) {
	account := connectionsapp.ConnectionInfo{Provider: "yahoo", ProviderAccountEmail: "user@yahoo.com"}
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
	account := connectionsapp.ConnectionInfo{Provider: "outlook", ProviderAccountEmail: "user@outlook.com"}
	err := classifySyncError(account, errors.Join(errors.New("payload too large"), errPayloadRejected))

	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		t.Fatalf("expected sync error type, got %T", err)
	}
	if syncErr.Code != appErrors.CodeSyncPayloadRejected {
		t.Fatalf("expected code %s, got %s", appErrors.CodeSyncPayloadRejected, syncErr.Code)
	}
}
