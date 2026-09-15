package legalentities

import (
	"net/http"

	legalentitiesEvents "github.com/bowerbird/internal/legalentities/adapters/events"
	httpV1 "github.com/bowerbird/internal/legalentities/adapters/http/v1"
	legalentitiesRepo "github.com/bowerbird/internal/legalentities/adapters/repository/postgres"
	"github.com/bowerbird/internal/legalentities/api"
	"github.com/bowerbird/internal/legalentities/application"
	"github.com/bowerbird/internal/legalentities/application/commands"
	"github.com/bowerbird/internal/legalentities/application/queries"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
)

func NewApplication(registry *database.Registry, eventBus events.EventBus) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	repo := legalentitiesRepo.NewRepository(registry)
	publisher := legalentitiesEvents.NewPublisher(eventBus)
	return &application.Application{
		Commands: application.Commands{
			CreateLegalEntity: commands.NewCreateLegalEntityCommand(repo, publisher),
			UpdateLegalEntity: commands.NewUpdateLegalEntityCommand(repo, publisher),
		},
		Queries: application.Queries{
			ListLegalEntities: queries.NewListLegalEntitiesQuery(repo),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("legalentities application is required")
	}
	controller := httpV1.NewController(app)
	httpV1.NewRouter(controller).Register(mux, cfg, authMiddleware)
}

func NewReceiverDirectory(app *application.Application) api.ReceiverDirectory {
	return application.NewReceiverDirectory(app)
}
