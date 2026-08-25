package tenant

import (
	"net/http"

	"github.com/bowerbird/internal/platform/config"
	httpV1 "github.com/bowerbird/internal/tenant/adapters/http/v1"
	provisionerpostgres "github.com/bowerbird/internal/tenant/adapters/provisioner/postgres"
	repositorypostgres "github.com/bowerbird/internal/tenant/adapters/repository/postgres"
	"github.com/bowerbird/internal/tenant/application"
	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewApplication(pool *pgxpool.Pool, databaseURL, migrationsDir string, defaults ports.DefaultPackApplier) *application.Application {
	if pool == nil {
		panic("control plane db pool is required")
	}
	if databaseURL == "" {
		panic("database url is required")
	}
	if migrationsDir == "" {
		panic("tenant migrations dir is required")
	}

	tenantRepo := repositorypostgres.NewPostgresRepository(pool)
	tenantProvisioner := provisionerpostgres.NewPostgresProvisioner(pool, databaseURL, migrationsDir)

	return application.NewApplication(tenantRepo, tenantProvisioner, defaults)
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("tenant application is required")
	}

	controller := httpV1.NewController(
		application.NewCreateTenantUseCaseFromCommand(app.Commands.CreateTenant),
		application.NewGetTenantUseCaseFromQuery(app.Queries.GetTenant),
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}
