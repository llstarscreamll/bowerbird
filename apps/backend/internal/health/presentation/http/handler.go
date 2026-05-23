package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/health/application"
	"github.com/money-path/bowerbird/apps/backend/internal/health/domain"
)

type Handler struct {
	useCase application.CheckHealthUseCase
}

func NewHandler(useCase application.CheckHealthUseCase) Handler {
	return Handler{useCase: useCase}
}

func (h Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/health", h.Health)
}

func (h Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	health := h.useCase.Execute(ctx)
	statusCode := http.StatusOK

	if health.Status == domain.StatusDegraded {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(health)
}
