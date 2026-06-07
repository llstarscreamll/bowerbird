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
		panic("organization controller is required")
	}

	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/organizations", authMiddleware(api.Wrap(h.controller.CreateOrganization, cfg)))
	mux.Handle("GET /api/v1/organizations/{id}", authMiddleware(api.Wrap(h.controller.GetOrganization, cfg)))
}
