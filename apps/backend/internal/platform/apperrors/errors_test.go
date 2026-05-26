package apperrors_test

import (
	"errors"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
)

func TestAppError_Error(t *testing.T) {
	err := apperrors.New(apperrors.CodeNotFound, "user not found")
	expected := "ERR_NOT_FOUND: user not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	wrapped := apperrors.Wrap(errors.New("db connection failed"), apperrors.CodeInternal, "internal server error")
	expectedWrapped := "ERR_INTERNAL: internal server error (db connection failed)"
	if wrapped.Error() != expectedWrapped {
		t.Errorf("expected %q, got %q", expectedWrapped, wrapped.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := apperrors.Wrap(baseErr, apperrors.CodeInternal, "failed")

	if !errors.Is(err, baseErr) {
		t.Errorf("expected wrapped error to match base error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("expected to be able to extract AppError")
	}
	if appErr.Code != apperrors.CodeInternal {
		t.Errorf("expected code %q, got %q", apperrors.CodeInternal, appErr.Code)
	}
}
