package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

func TestMapError_AppError(t *testing.T) {
	appErr := apperrors.New(apperrors.CodeNotFound, "resource missing")
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
	if errObj.Code != apperrors.CodeNotFound {
		t.Errorf("expected code %q, got %q", apperrors.CodeNotFound, errObj.Code)
	}
	if errObj.Detail != "resource missing" {
		t.Errorf("expected detail %q, got %q", "resource missing", errObj.Detail)
	}
	if errObj.Meta["_debug"] != nil {
		t.Errorf("expected no _debug in production mode")
	}
}

func TestMapError_GenericError_DevMode(t *testing.T) {
	genericErr := errors.New("db query failed")
	doc, status := api.MapError(genericErr, "trace-456", true)

	if status != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, status)
	}

	errObj := doc.Errors[0]
	if errObj.Code != apperrors.CodeInternal {
		t.Errorf("expected code %q, got %q", apperrors.CodeInternal, errObj.Code)
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

func TestWrap(t *testing.T) {
	expectedErr := apperrors.New(apperrors.CodeValidation, "invalid input")
	handlerFunc := func(w http.ResponseWriter, r *http.Request) error {
		return expectedErr
	}

	wrapped := api.Wrap(handlerFunc, false)

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

	if len(doc.Errors) != 1 || doc.Errors[0].Code != apperrors.CodeValidation {
		t.Errorf("expected json:api response to contain the validation error")
	}
}
