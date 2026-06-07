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
		panic("connections controller is required")
	}

	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/connections", authMiddleware(api.Wrap(h.controller.ListConnections, cfg)))
	mux.Handle("GET /api/v1/connections/google", authMiddleware(api.Wrap(h.controller.GoogleConnect, cfg)))
	mux.Handle("GET /api/v1/connections/google/callback", api.Wrap(h.controller.GoogleCallback, cfg))
	mux.Handle("DELETE /api/v1/connections/{id}", authMiddleware(api.Wrap(h.controller.DeleteConnection, cfg)))
}
