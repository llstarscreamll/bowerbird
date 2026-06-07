package errors_test

import (
	"errors"
	"testing"

	appErrors "github.com/bowerbird/internal/platform/errors"
)

func TestSyncError_UXMeta(t *testing.T) {
	err := appErrors.NewSync(
		appErrors.CodeSyncReauthRequired,
		"requires reauth",
		appErrors.SyncErrorOptions{
			Provider:       "GMAIL",
			AccountEmail:   "user@gmail.com",
			RequiresReauth: true,
			Meta: map[string]any{
				"custom": "value",
			},
		},
	)

	meta := err.UXMeta()
	if meta["provider"] != "GMAIL" {
		t.Fatalf("expected provider metadata")
	}
	if meta["account_email"] != "user@gmail.com" {
		t.Fatalf("expected account_email metadata")
	}
	if meta["requires_reauth"] != true {
		t.Fatalf("expected requires_reauth metadata")
	}
	if meta["custom"] != "value" {
		t.Fatalf("expected custom metadata")
	}
}

func TestWrapSync_Unwrap(t *testing.T) {
	baseErr := errors.New("provider failed")
	err := appErrors.WrapSync(baseErr, appErrors.CodeSyncInternal, "sync failed", appErrors.SyncErrorOptions{})

	if !errors.Is(err, baseErr) {
		t.Fatalf("expected wrapped sync error to unwrap base error")
	}
}

func TestHelpURLForCode(t *testing.T) {
	url := appErrors.HelpURLForCode(appErrors.CodeSyncRateLimited)
	if url == "" {
		t.Fatalf("expected help URL for sync error code")
	}
}
