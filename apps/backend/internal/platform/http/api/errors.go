package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
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
	Links  *ErrorLinks    `json:"links,omitempty"`
	Source *ErrorSource   `json:"source,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type ErrorLinks struct {
	About string `json:"about,omitempty"`
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

	var appErr *appErrors.AppError
	var syncErr *appErrors.SyncError
	var httpStatus int
	var title, detail, code, helpURL string
	meta := map[string]any{}

	if errors.As(err, &syncErr) {
		code = syncErr.Code
		detail = syncErr.Message
		httpStatus, title = statusFromCode(syncErr.Code)
		helpURL = syncErr.HelpURL
		if helpURL == "" {
			helpURL = appErrors.HelpURLForCode(syncErr.Code)
		}
		for key, value := range filterAllowedErrorMeta(syncErr.UXMeta()) {
			meta[key] = value
		}
	} else if errors.As(err, &appErr) {
		code = appErr.Code
		detail = appErr.Message
		httpStatus, title = statusFromCode(appErr.Code)
		helpURL = appErrors.HelpURLForCode(appErr.Code)
	} else {
		// Generic or unexpected error
		code = appErrors.CodeInternal
		detail = "An unexpected error occurred."
		httpStatus = http.StatusInternalServerError
		title = http.StatusText(http.StatusInternalServerError)
	}

	detail = redactSensitive(detail)

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

	if helpURL != "" {
		obj.Links = &ErrorLinks{About: helpURL}
	}

	for key, value := range meta {
		obj.Meta[key] = redactAnySensitive(value)
	}

	// In development, populate _debug
	if isDev {
		obj.Meta["_debug"] = map[string]any{
			"error":       redactSensitive(err.Error()),
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
	case appErrors.CodeSyncReauthRequired:
		return http.StatusUnauthorized, "Unauthorized"
	case appErrors.CodeSyncRateLimited:
		return http.StatusTooManyRequests, "Too Many Requests"
	case appErrors.CodeSyncProviderTemporary:
		return http.StatusServiceUnavailable, "Service Unavailable"
	case appErrors.CodeSyncPayloadRejected:
		return http.StatusUnprocessableEntity, "Unprocessable Entity"
	case appErrors.CodeSyncInternal:
		return http.StatusInternalServerError, "Internal Server Error"
	case appErrors.CodeValidation:
		return http.StatusBadRequest, "Bad Request"
	case appErrors.CodeUnauthorized:
		return http.StatusUnauthorized, "Unauthorized"
	case appErrors.CodeForbidden:
		return http.StatusForbidden, "Forbidden"
	case appErrors.CodeNotFound:
		return http.StatusNotFound, "Not Found"
	case appErrors.CodeConflict:
		return http.StatusConflict, "Conflict"
	case appErrors.CodeNotImplemented:
		return http.StatusNotImplemented, "Not Implemented"
	case appErrors.CodeInternal:
		fallthrough
	default:
		return http.StatusInternalServerError, "Internal Server Error"
	}
}

func filterAllowedErrorMeta(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return nil
	}

	allowed := map[string]struct{}{
		"requires_reauth":     {},
		"provider":            {},
		"retry_after_seconds": {},
		"account_email":       {},
	}

	out := map[string]any{}
	for key, value := range meta {
		if _, ok := allowed[key]; ok {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}

	return out
}

var sensitiveValuePattern = regexp.MustCompile(`(?i)(access[_-]?token|refresh[_-]?token|password|authorization)\s*[:=]\s*([^\s,;]+)`)
var bearerTokenPattern = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-\._~\+/]+=*`)
var authorizationBearerPattern = regexp.MustCompile(`(?i)(authorization)\s*[:=]\s*bearer\s+[^\s,;]+`)

func redactSensitive(text string) string {
	if text == "" {
		return text
	}

	redacted := bearerTokenPattern.ReplaceAllString(text, "Bearer [REDACTED]")
	redacted = authorizationBearerPattern.ReplaceAllString(redacted, "$1=[REDACTED]")
	redacted = sensitiveValuePattern.ReplaceAllString(redacted, "$1=[REDACTED]")
	redacted = bearerTokenPattern.ReplaceAllString(redacted, "Bearer [REDACTED]")
	return redacted
}

func redactAnySensitive(value any) any {
	strValue, ok := value.(string)
	if !ok {
		return value
	}

	return redactSensitive(strValue)
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
