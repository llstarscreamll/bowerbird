package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
)

// JSONAPIErrorDocument is the top-level structure for JSON:API errors.
type JSONAPIErrorDocument struct {
	Errors []JSONAPIErrorObject `json:"errors"`
}

// JSONAPIErrorObject represents a single error in JSON:API format.
type JSONAPIErrorObject struct {
	ID     string         `json:"id,omitempty"`
	Status string         `json:"status,omitempty"`
	Code   string         `json:"code,omitempty"`
	Title  string         `json:"title,omitempty"`
	Detail string         `json:"detail,omitempty"`
	Source *ErrorSource   `json:"source,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

// MapError translates a standard Go error into a JSON:API error response and status code.
func MapError(err error, traceID string, isDev bool) (JSONAPIErrorDocument, int) {
	doc := JSONAPIErrorDocument{
		Errors: []JSONAPIErrorObject{},
	}

	var appErr *apperrors.AppError
	var httpStatus int
	var title, detail, code string

	if errors.As(err, &appErr) {
		code = appErr.Code
		detail = appErr.Message
		httpStatus, title = statusFromCode(appErr.Code)
	} else {
		// Generic or unexpected error
		code = apperrors.CodeInternal
		detail = "An unexpected error occurred."
		httpStatus = http.StatusInternalServerError
		title = http.StatusText(http.StatusInternalServerError)
	}

	obj := JSONAPIErrorObject{
		ID:     traceID,
		Status: fmt.Sprintf("%d", httpStatus),
		Code:   code,
		Title:  title,
		Detail: detail,
		Meta: map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	// In development, populate _debug
	if isDev {
		obj.Meta["_debug"] = map[string]any{
			"error":       err.Error(),
			"stack_trace": captureStackTrace(3), // skip up to the handler
		}
	}

	doc.Errors = append(doc.Errors, obj)
	return doc, httpStatus
}

// RespondWithError writes an error response adhering to JSON:API.
func RespondWithError(w http.ResponseWriter, r *http.Request, err error, isDev bool) {
	// Extract or generate trace ID (using sentry-trace header if present)
	traceID := r.Header.Get("sentry-trace")
	if traceID == "" {
		// Basic generation for now. Should use a proper generator.
		traceID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	doc, status := MapError(err, traceID, isDev)

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(doc)
}

func statusFromCode(code string) (int, string) {
	switch code {
	case apperrors.CodeValidation:
		return http.StatusBadRequest, "Bad Request"
	case apperrors.CodeUnauthorized:
		return http.StatusUnauthorized, "Unauthorized"
	case apperrors.CodeForbidden:
		return http.StatusForbidden, "Forbidden"
	case apperrors.CodeNotFound:
		return http.StatusNotFound, "Not Found"
	case apperrors.CodeConflict:
		return http.StatusConflict, "Conflict"
	case apperrors.CodeNotImplemented:
		return http.StatusNotImplemented, "Not Implemented"
	case apperrors.CodeInternal:
		fallthrough
	default:
		return http.StatusInternalServerError, "Internal Server Error"
	}
}

func captureStackTrace(skip int) string {
	var builder strings.Builder
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "No stack trace available"
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// Only include our own app frames for brevity, unless debugging framework
		if strings.Contains(frame.File, "bowerbird") {
			builder.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		}
		if !more {
			break
		}
	}
	return builder.String()
}

// GenerateTraceIDMiddleware adds a trace ID to the context if not present (Optional helper)
func GenerateTraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("sentry-trace")
		if traceID == "" {
			traceID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			r.Header.Set("sentry-trace", traceID)
		}
		ctx := context.WithValue(r.Context(), "traceID", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
