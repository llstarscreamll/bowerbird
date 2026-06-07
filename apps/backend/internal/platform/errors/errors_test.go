package errors_test

import (
	"errors"
	"testing"

	appErrors "github.com/bowerbird/internal/platform/errors"
)

func TestAppError_Error(t *testing.T) {
	err := appErrors.New(appErrors.CodeNotFound, "user not found")
	expected := "ERR_NOT_FOUND: user not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	wrapped := appErrors.Wrap(errors.New("db connection failed"), appErrors.CodeInternal, "internal server error")
	expectedWrapped := "ERR_INTERNAL: internal server error (db connection failed)"
	if wrapped.Error() != expectedWrapped {
		t.Errorf("expected %q, got %q", expectedWrapped, wrapped.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := appErrors.Wrap(baseErr, appErrors.CodeInternal, "failed")

	if !errors.Is(err, baseErr) {
		t.Errorf("expected wrapped error to match base error")
	}

	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("expected to be able to extract AppError")
	}
	if appErr.Code != appErrors.CodeInternal {
		t.Errorf("expected code %q, got %q", appErrors.CodeInternal, appErr.Code)
	}
}
