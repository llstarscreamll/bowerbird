package organization

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/organization/adapters/http/v1"
	provisionerpostgres "github.com/bowerbird/internal/organization/adapters/provisioner/postgres"
	repositorypostgres "github.com/bowerbird/internal/organization/adapters/repository/postgres"
	"github.com/bowerbird/internal/organization/application"
	"github.com/bowerbird/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewApplication(pool *pgxpool.Pool, databaseURL, migrationsDir string) *application.Application {
	if pool == nil {
		panic("control plane db pool is required")
	}
	if databaseURL == "" {
		panic("database url is required")
	}
	if migrationsDir == "" {
		panic("tenant migrations dir is required")
	}

	organizationRepo := repositorypostgres.NewPostgresRepository(pool)
	organizationProvisioner := provisionerpostgres.NewPostgresProvisioner(pool, databaseURL, migrationsDir)

	return application.NewApplication(organizationRepo, organizationProvisioner)
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("organization application is required")
	}

	controller := httpV1.NewController(
		application.NewCreateOrganizationUseCaseFromCommand(app.Commands.CreateOrganization),
		application.NewGetOrganizationUseCaseFromQuery(app.Queries.GetOrganization),
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}
