package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bowerbird/internal/platform/config"
)

// HandlerFunc defines the signature for an API handler that returns an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Wrap converts an API HandlerFunc into a standard http.HandlerFunc.
// It intercepts errors returned by the handler and formats them using the JSON:API standard.
func Wrap(h HandlerFunc, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			if cfg.Debug {
				log.Printf("[API Error] %s %s: %v", r.Method, r.URL.Path, err)
			}
			RespondWithError(w, r, err, cfg.Debug)
		}
	}
}

// Success writes a standard JSON:API successful response (optional helper for consistency).
func Success(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)

	if data != nil {
		return json.NewEncoder(w).Encode(data)
	}

	return nil
}
