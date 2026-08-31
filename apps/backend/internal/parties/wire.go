package parties

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/parties/adapters/http/v1"
	partiesRepo "github.com/bowerbird/internal/parties/adapters/repository/postgres"
	"github.com/bowerbird/internal/parties/application"
	"github.com/bowerbird/internal/parties/application/commands"
	"github.com/bowerbird/internal/parties/application/queries"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
)

func NewApplication(registry *database.Registry) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	repo := partiesRepo.NewPartyRepository(registry)
	return &application.Application{
		Commands: application.Commands{
			ResolveOrCreateFromIssuer: commands.NewResolveOrCreateFromIssuerCommand(repo),
			CreateParty:               commands.NewCreatePartyCommand(repo),
			UpdateParty:               commands.NewUpdatePartyCommand(repo),
		},
		Queries: application.Queries{
			GetPartyByID: queries.NewGetPartyByIDQuery(repo),
			ListParties:  queries.NewListPartiesQuery(repo),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("parties application is required")
	}
	controller := httpV1.NewController(app)
	httpV1.NewRouter(controller).Register(mux, cfg, authMiddleware)
}

func NewIssuerPartyLookup(app *application.Application) application.IssuerPartyLookup {
	return application.NewIssuerPartyLookupFromApp(app)
}
