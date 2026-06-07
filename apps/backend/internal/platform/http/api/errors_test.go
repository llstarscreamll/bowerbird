package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

func TestMapError_AppError(t *testing.T) {
	appErr := appErrors.New(appErrors.CodeNotFound, "resource missing")
	doc, status := api.MapError(appErr, "trace-123", false)

	if status != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, status)
	}
	if len(doc.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(doc.Errors))
	}

	errObj := doc.Errors[0]
	if errObj.ID != "trace-123" {
		t.Errorf("expected trace-123, got %q", errObj.ID)
	}
	if errObj.Status != "404" {
		t.Errorf("expected status 404, got %q", errObj.Status)
	}
	if errObj.Code != appErrors.CodeNotFound {
		t.Errorf("expected code %q, got %q", appErrors.CodeNotFound, errObj.Code)
	}
	if errObj.Detail != "resource missing" {
		t.Errorf("expected detail %q, got %q", "resource missing", errObj.Detail)
	}
	if errObj.Meta["_debug"] != nil {
		t.Errorf("expected no _debug in production mode")
	}
	if errObj.Links != nil {
		t.Errorf("expected no links for generic app error, got %+v", errObj.Links)
	}
}

func TestMapError_GenericError_DevMode(t *testing.T) {
	genericErr := errors.New("db query failed")
	doc, status := api.MapError(genericErr, "trace-456", true)

	if status != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, status)
	}

	errObj := doc.Errors[0]
	if errObj.Code != appErrors.CodeInternal {
		t.Errorf("expected code %q, got %q", appErrors.CodeInternal, errObj.Code)
	}

	debugInfo, ok := errObj.Meta["_debug"].(map[string]any)
	if !ok {
		t.Fatalf("expected _debug to be a map, got %T", errObj.Meta["_debug"])
	}
	if debugInfo["error"] != "db query failed" {
		t.Errorf("expected debug error to match original error")
	}
	if debugInfo["stack_trace"] == "" {
		t.Errorf("expected stack trace to be populated")
	}
}

func TestMapError_SyncError_MetadataAndHelpLink(t *testing.T) {
	syncErr := appErrors.NewSync(
		appErrors.CodeSyncRateLimited,
		"La cuenta de Yahoo personal@yahoo.com requiere espera temporal",
		appErrors.SyncErrorOptions{
			Provider:          "YAHOO",
			AccountEmail:      "personal@yahoo.com",
			RetryAfterSeconds: 120,
			Meta: map[string]any{
				"requires_reauth": false,
				"internal_note":   "should not leak",
			},
		},
	)

	doc, status := api.MapError(syncErr, "trace-789", false)

	if status != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, status)
	}

	errObj := doc.Errors[0]
	if errObj.Links == nil || errObj.Links.About == "" {
		t.Fatalf("expected links.about to be set")
	}
	if errObj.Links.About != "https://help.bowerbird.dev/errors/ERR_SYNC_RATE_LIMITED" {
		t.Fatalf("unexpected links.about: %s", errObj.Links.About)
	}
	if errObj.Meta["provider"] != "YAHOO" {
		t.Fatalf("expected provider metadata, got %v", errObj.Meta["provider"])
	}
	if errObj.Meta["retry_after_seconds"] != 120 {
		t.Fatalf("expected retry_after_seconds=120, got %v", errObj.Meta["retry_after_seconds"])
	}
	if errObj.Meta["internal_note"] != nil {
		t.Fatalf("unexpected leaked internal_note metadata: %v", errObj.Meta["internal_note"])
	}
}

func TestMapError_RedactsSensitiveInfoInDevDebug(t *testing.T) {
	err := errors.New("authorization=Bearer abc123token password=super-secret")
	doc, _ := api.MapError(err, "trace-999", true)

	debugInfo, ok := doc.Errors[0].Meta["_debug"].(map[string]any)
	if !ok {
		t.Fatalf("expected _debug to be a map")
	}

	debugError, ok := debugInfo["error"].(string)
	if !ok {
		t.Fatalf("expected debug error to be string")
	}

	if debugError == "authorization=Bearer abc123token password=super-secret" {
		t.Fatalf("expected debug error to be redacted")
	}
	if containsSensitive(debugError) {
		t.Fatalf("debug error still contains sensitive info: %s", debugError)
	}
}

func containsSensitive(text string) bool {
	for _, forbidden := range []string{"abc123token", "super-secret"} {
		if strings.Contains(text, forbidden) {
			return true
		}
	}
	return false
}

func TestWrap(t *testing.T) {
	expectedErr := appErrors.New(appErrors.CodeValidation, "invalid input")
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	wrapped := api.Wrap(handlerFunc, config.Config{Debug: false})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("expected json:api content type")
	}

	var doc api.JSONAPIErrorDocument
	if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(doc.Errors) != 1 || doc.Errors[0].Code != appErrors.CodeValidation {
		t.Errorf("expected json:api response to contain the validation error")
	}
}
