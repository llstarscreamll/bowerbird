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
		panic("tenant controller is required")
	}

	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/tenants", authMiddleware(api.Wrap(h.controller.CreateTenant, cfg)))
	mux.Handle("GET /api/v1/tenants/{id}", authMiddleware(api.Wrap(h.controller.GetTenant, cfg)))
}
