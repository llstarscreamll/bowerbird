package v1

import (
	"net/http"

	identityapi "github.com/bowerbird/internal/identity/api"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Router struct {
	controller *Controller
	operators  identityapi.OperatorDirectory
}

func NewRouter(controller *Controller, operators identityapi.OperatorDirectory) *Router {
	if controller == nil {
		panic("entitlements controller is required")
	}
	if operators == nil {
		panic("operator directory is required")
	}
	return &Router{controller: controller, operators: operators}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/entitlements", authMiddleware(api.Wrap(h.controller.GetEntitlements, cfg)))

	operator := func(handler api.HandlerFunc) http.Handler {
		return authMiddleware(h.requireOperator(cfg, api.Wrap(handler, cfg)))
	}
	mux.Handle("GET /api/v1/platform/tenants", operator(h.controller.ListPlatformTenants))
	mux.Handle("GET /api/v1/platform/tenants/{id}/entitlements", operator(h.controller.GetPlatformTenantEntitlements))
	mux.Handle("PUT /api/v1/platform/tenants/{id}/entitlements", operator(h.controller.PutPlatformTenantEntitlements))
}

func (h *Router) requireOperator(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			api.RespondWithError(w, r, appErrors.New(appErrors.CodeUnauthorized, "unauthorized"), cfg.Debug)
			return
		}

		isOperator, err := h.operators.IsPlatformOperator(r.Context(), claims.UserID)
		if err != nil {
			api.RespondWithError(w, r, appErrors.Wrap(err, appErrors.CodeInternal, "failed to resolve platform role"), cfg.Debug)
			return
		}
		if !isOperator {
			api.RespondWithError(w, r, appErrors.New(appErrors.CodeForbidden, "operator access required"), cfg.Debug)
			return
		}

		next.ServeHTTP(w, r)
	})
}
