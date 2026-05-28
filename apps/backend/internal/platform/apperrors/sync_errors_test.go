package apperrors_test

import (
	"errors"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
)

func TestSyncError_UXMeta(t *testing.T) {
	err := apperrors.NewSync(
		apperrors.CodeSyncReauthRequired,
		"requires reauth",
		apperrors.SyncErrorOptions{
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
	err := apperrors.WrapSync(baseErr, apperrors.CodeSyncInternal, "sync failed", apperrors.SyncErrorOptions{})

	if !errors.Is(err, baseErr) {
		t.Fatalf("expected wrapped sync error to unwrap base error")
	}
}

func TestHelpURLForCode(t *testing.T) {
	url := apperrors.HelpURLForCode(apperrors.CodeSyncRateLimited)
	if url == "" {
		t.Fatalf("expected help URL for sync error code")
	}
}
