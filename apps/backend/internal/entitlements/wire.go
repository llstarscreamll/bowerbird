package entitlements

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/entitlements/adapters/http/v1"
	entitlementsPostgres "github.com/bowerbird/internal/entitlements/adapters/repository/postgres"
	tenantDir "github.com/bowerbird/internal/entitlements/adapters/tenants"
	"github.com/bowerbird/internal/entitlements/application"
	orgpostgres "github.com/bowerbird/internal/organization/adapters/repository/postgres"
	"github.com/bowerbird/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewApplication(pool *pgxpool.Pool) *application.Service {
	if pool == nil {
		panic("control plane db pool is required")
	}
	return application.NewService(entitlementsPostgres.NewRepository(pool))
}

func NewHTTPHandler(
	mux *http.ServeMux,
	svc *application.Service,
	operators application.OperatorDirectory,
	pool *pgxpool.Pool,
	authMiddleware func(http.Handler) http.Handler,
	cfg config.Config,
) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if svc == nil {
		panic("entitlements service is required")
	}
	if operators == nil {
		panic("operator directory is required")
	}
	if pool == nil {
		panic("control plane db pool is required")
	}

	controller := httpV1.NewController(svc, tenantDir.NewDirectory(orgpostgres.NewPostgresRepository(pool)))
	router := httpV1.NewRouter(controller, operators)
	router.Register(mux, cfg, authMiddleware)
	return router
}
