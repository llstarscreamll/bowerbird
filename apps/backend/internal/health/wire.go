package health

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/health/adapters/http/v1"
	healthRepo "github.com/bowerbird/internal/health/adapters/repository/postgres"
	"github.com/bowerbird/internal/health/application"
	"github.com/bowerbird/internal/health/application/queries"
	"github.com/bowerbird/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewApplication(db *pgxpool.Pool) *application.Application {
	if db == nil {
		panic("database pool is required")
	}

	repo := healthRepo.NewPostgresRepository(db)

	return &application.Application{
		Queries: application.Queries{
			CheckHealth: queries.NewCheckHealthQuery(repo),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}

	if app == nil {
		panic("health application is required")
	}

	controller := httpV1.NewController(*app)
	handler := httpV1.NewRouter(controller)
	handler.Register(mux, cfg)

	return handler
}
