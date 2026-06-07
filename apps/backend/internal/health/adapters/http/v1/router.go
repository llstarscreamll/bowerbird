package v1

import (
	"net/http"

	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/http/api"
)

type Router struct {
	controller *Controller
}

func NewRouter(controller *Controller) *Router {
	if controller == nil {
		panic("health controller is required")
	}

	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config) {
	mux.Handle("GET /health", api.Wrap(h.controller.Health, cfg))
	mux.Handle("GET /api/health", api.Wrap(h.controller.Health, cfg))
}
