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
		panic("inbox controller is required")
	}

	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/inbox/sync-status", authMiddleware(api.Wrap(h.controller.ListAccountSyncStatus, cfg)))
	mux.Handle("GET /api/v1/inbox/messages", authMiddleware(api.Wrap(h.controller.ListMessages, cfg)))
	mux.Handle("GET /api/v1/inbox/messages/{messageID}", authMiddleware(api.Wrap(h.controller.GetMessage, cfg)))
	mux.Handle("POST /api/v1/inbox/sync", authMiddleware(api.Wrap(h.controller.Sync, cfg)))
}
