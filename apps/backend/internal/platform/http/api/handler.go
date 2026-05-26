package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// HandlerFunc defines the signature for an API handler that returns an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Wrap converts an API HandlerFunc into a standard http.HandlerFunc.
// It intercepts errors returned by the handler and formats them using the JSON:API standard.
func Wrap(h HandlerFunc, isDev bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			if isDev {
				log.Printf("[API Error] %s %s: %v", r.Method, r.URL.Path, err)
			}
			RespondWithError(w, r, err, isDev)
		}
	}
}

// Success writes a standard JSON:API successful response (optional helper for consistency).
// We are primarily focusing on errors, but keeping standard JSON responses here is good.
func Success(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	// Optionally wrap data in {"data": ...} if it's not already wrapped
	// depending on how strict we are with JSON:API for success responses.
	// For now, we will assume `data` is properly formatted.
	if data != nil {
		return json.NewEncoder(w).Encode(data)
	}
	return nil
}
