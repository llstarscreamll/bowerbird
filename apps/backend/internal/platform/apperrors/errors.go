package apperrors

import (
	"fmt"
)

// AppError represents a domain error with a unique code and an underlying cause.
type AppError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap allows errors.Is and errors.As to work with the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError with a code and a message.
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap creates a new AppError wrapping an existing error.
func Wrap(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes (Examples, can be extended by domain packages)
const (
	CodeInternal       = "ERR_INTERNAL"
	CodeNotFound       = "ERR_NOT_FOUND"
	CodeValidation     = "ERR_VALIDATION"
	CodeUnauthorized   = "ERR_UNAUTHORIZED"
	CodeForbidden      = "ERR_FORBIDDEN"
	CodeConflict       = "ERR_CONFLICT"
	CodeNotImplemented = "ERR_NOT_IMPLEMENTED"
)
