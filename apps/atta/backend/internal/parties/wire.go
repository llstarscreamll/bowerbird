package parties

import (
	"net/http"

	httpV1 "github.com/atta/internal/parties/adapters/http/v1"
	partiesRepo "github.com/atta/internal/parties/adapters/repository/postgres"
	"github.com/atta/internal/parties/api"
	"github.com/atta/internal/parties/application"
	"github.com/atta/internal/parties/application/commands"
	"github.com/atta/internal/parties/application/queries"
	"github.com/atta/internal/platform/config"
	"github.com/atta/internal/platform/database"
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
			PartyChannels:             commands.NewPartyChannelsCommand(repo),
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

func NewIssuerPartyLookup(app *application.Application) api.IssuerPartyLookup {
	return application.NewIssuerPartyLookupFromApp(app)
}
