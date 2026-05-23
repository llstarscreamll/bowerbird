package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/money-path/turno/apps/api/internal/repository"
)

type HTTPHandler struct {
	healthRepo repository.HealthRepository
}

func NewHTTPHandler(healthRepo repository.HealthRepository) HTTPHandler {
	return HTTPHandler{healthRepo: healthRepo}
}

func (h HTTPHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/health", h.Health)
}

func (h HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	statusCode := http.StatusOK

	if err := h.healthRepo.IsDatabaseHealthy(ctx); err != nil {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]string{"status": status}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
