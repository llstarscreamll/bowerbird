package http

import (
	"context"
	"net/http"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/health/application"
	"github.com/money-path/bowerbird/apps/backend/internal/health/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	useCase application.CheckHealthUseCase
}

func NewHandler(useCase application.CheckHealthUseCase) Handler {
	return Handler{useCase: useCase}
}

func (h Handler) Register(mux *http.ServeMux, isDev bool) {
	mux.HandleFunc("GET /health", api.Wrap(h.Health, isDev))
	mux.HandleFunc("GET /api/health", api.Wrap(h.Health, isDev))
}

func (h Handler) Health(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	health := h.useCase.Execute(ctx)
	statusCode := http.StatusOK

	if health.Status == domain.StatusDegraded {
		statusCode = http.StatusServiceUnavailable
	}

	return api.Success(w, statusCode, health)
}
