package catalog

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/catalog/adapters/http/v1"
	"github.com/bowerbird/internal/catalog/adapters/matchers"
	catalogRepo "github.com/bowerbird/internal/catalog/adapters/repository/postgres"
	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/queries"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
)

func NewApplication(registry *database.Registry) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	repo := catalogRepo.NewCatalogRepository(registry)
	matcher := matchers.NewNormalizedDescriptionMatcher(repo)
	remember := commands.NewRememberDecisionCommand(repo, repo, repo, repo)
	return &application.Application{
		Commands: application.Commands{
			ResolveInvoiceLine: commands.NewResolveInvoiceLineCommand(repo, repo, repo, matcher),
			RememberDecision:   remember,
			LinkInvoiceLine:    commands.NewLinkInvoiceLineCommand(remember),
		},
		Queries: application.Queries{
			GetItemByID:     queries.NewGetItemByIDQuery(repo),
			GetItemNames:    queries.NewGetItemNamesQuery(repo),
			ListItems:       queries.NewListItemsQuery(repo),
			ListReviewQueue: queries.NewListReviewQueueQuery(repo),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("catalog application is required")
	}
	httpV1.NewRouter(httpV1.NewController(app)).Register(mux, cfg, authMiddleware)
}
