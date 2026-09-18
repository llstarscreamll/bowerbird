package entitlements

import (
	"net/http"

	httpV1 "github.com/atta/internal/entitlements/adapters/http/v1"
	entitlementsPostgres "github.com/atta/internal/entitlements/adapters/repository/postgres"
	tenantDir "github.com/atta/internal/entitlements/adapters/tenants"
	"github.com/atta/internal/entitlements/api"
	"github.com/atta/internal/entitlements/application"
	identityapi "github.com/atta/internal/identity/api"
	"github.com/atta/internal/platform/config"
	tenantapi "github.com/atta/internal/tenant/api"
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
	operators identityapi.OperatorDirectory,
	tenants tenantapi.Directory,
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
	if tenants == nil {
		panic("tenant directory is required")
	}

	controller := httpV1.NewController(svc, tenantDir.NewDirectory(tenants))
	router := httpV1.NewRouter(controller, operators)
	router.Register(mux, cfg, authMiddleware)
	return router
}

var (
	_ api.Features    = (*application.Service)(nil)
	_ api.DefaultPack = (*application.Service)(nil)
)
