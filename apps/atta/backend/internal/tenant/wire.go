package tenant

import (
	"net/http"

	"github.com/atta/internal/platform/config"
	httpV1 "github.com/atta/internal/tenant/adapters/http/v1"
	provisionerpostgres "github.com/atta/internal/tenant/adapters/provisioner/postgres"
	repositorypostgres "github.com/atta/internal/tenant/adapters/repository/postgres"
	"github.com/atta/internal/tenant/api"
	"github.com/atta/internal/tenant/application"
	"github.com/atta/internal/tenant/application/ports"
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
	tenantProvisioner := provisionerpostgres.NewPostgresProvisioner(databaseURL, migrationsDir)

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
		app.Commands.CreateTenant,
		app.Queries.GetTenant,
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}

func NewDirectory(app *application.Application) api.Directory {
	if app == nil {
		panic("tenant application is required")
	}
	return app.Directory()
}
