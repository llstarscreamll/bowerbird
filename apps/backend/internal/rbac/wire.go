package rbac

import (
	"net/http"

	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	httpV1 "github.com/bowerbird/internal/rbac/adapters/http/v1"
	rbacRepo "github.com/bowerbird/internal/rbac/adapters/repository/postgres"
	"github.com/bowerbird/internal/rbac/application"
)

func NewService(registry *database.Registry) *application.Service {
	if registry == nil {
		panic("database registry is required")
	}
	return application.NewService(rbacRepo.NewPermissionRepository(registry))
}

func NewHTTPHandler(mux *http.ServeMux, svc *application.Service, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	if mux == nil {
		panic("http mux is required")
	}
	if svc == nil {
		panic("rbac service is required")
	}
	controller := httpV1.NewController(svc)
	httpV1.NewRouter(controller).Register(mux, cfg, authMiddleware)
}
