package v1

import (
	"net/http"

	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"github.com/bowerbird/internal/rbac/application"
)

type Controller struct {
	svc *application.Service
}

func NewController(svc *application.Service) *Controller {
	if svc == nil {
		panic("rbac service is required")
	}
	return &Controller{svc: svc}
}

type permissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func (c *Controller) GetMyPermissions(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	codes, err := c.svc.ListEffectivePermissions(r.Context(), claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list permissions")
	}

	return api.Success(w, http.StatusOK, permissionsResponse{Permissions: codes})
}

type Router struct {
	controller *Controller
}

func NewRouter(controller *Controller) *Router {
	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/rbac/me/permissions", authMiddleware(api.Wrap(h.controller.GetMyPermissions, cfg)))
}
