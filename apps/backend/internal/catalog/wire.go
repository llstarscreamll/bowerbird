package catalog

import (
	"net/http"

	httpV1 "github.com/bowerbird/internal/catalog/adapters/http/v1"
	catalogJobs "github.com/bowerbird/internal/catalog/adapters/jobs"
	"github.com/bowerbird/internal/catalog/adapters/matchers"
	catalogRepo "github.com/bowerbird/internal/catalog/adapters/repository/postgres"
	"github.com/bowerbird/internal/catalog/api"
	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/application/queries"
	contractJobs "github.com/bowerbird/internal/catalog/contracts/jobs"
	filesapi "github.com/bowerbird/internal/files/api"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/scheduler"
)

func NewApplication(registry *database.Registry, jobQueue jobs.TaskQueue, objects filesapi.TenantObjects) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}
	if objects == nil {
		panic("tenant objects are required")
	}
	repo := catalogRepo.NewCatalogRepository(registry)
	matcher := matchers.NewNormalizedDescriptionMatcher(repo)
	return &application.Application{
		Commands: application.Commands{
			ResolveInvoiceLine:          commands.NewResolveInvoiceLineCommand(repo, repo, repo, repo, matcher),
			ValidateCatalogItem:         commands.NewValidateCatalogItemCommand(repo),
			MintProvisionalFromEvidence: commands.NewMintProvisionalFromEvidenceCommand(repo, repo, repo),
			EnsureSupplierAlias:         commands.NewEnsureSupplierAliasCommand(repo),
			RecordMatchMemory:           commands.NewRecordMatchMemoryCommand(repo),
			CreateItem:                  commands.NewCreateItemCommand(repo),
			UpdateItem:                  commands.NewUpdateItemCommand(repo),
			QueueCatalogImport:          commands.NewQueueCatalogImportCommand(repo, objects, jobQueue),
			ProcessCatalogImport:        commands.NewProcessCatalogImportCommand(repo, repo, objects, jobQueue),
			CancelCatalogImport:         commands.NewCancelCatalogImportCommand(repo),
			PurgeStaleCatalogImports:    commands.NewPurgeStaleCatalogImportsCommand(repo),
		},
		Queries: application.Queries{
			GetItemByID:      queries.NewGetItemByIDQuery(repo),
			GetItemNames:     queries.NewGetItemNamesQuery(repo),
			GetItemDisplays:  queries.NewGetItemDisplaysQuery(repo),
			ListItems:        queries.NewListItemsQuery(repo),
			GetImportByID:    queries.NewGetImportByIDQuery(repo),
			GetActiveImport:  queries.NewGetActiveImportQuery(repo),
			ListImports:      queries.NewListImportsQuery(repo),
			ListImportErrors: queries.NewListImportErrorsQuery(repo),
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

func RegisterJobs(app *application.Application, tenants ports.ActiveTenantLister) []jobs.JobHandler {
	if app == nil {
		panic("catalog application is required")
	}
	if tenants == nil {
		panic("tenant lister is required")
	}
	return []jobs.JobHandler{
		catalogJobs.NewProcessCatalogImport(app.Commands.ProcessCatalogImport),
		catalogJobs.NewProcessCatalogImportPurge(
			commands.NewPlatformPurgeStaleCatalogImportsCommand(app.Commands.PurgeStaleCatalogImports, tenants),
		),
	}
}

func RegisterSchedules() []scheduler.Rule {
	return []scheduler.Rule{{
		Name:     "catalog-import-purge",
		Schedule: "0 5 * * *",
		JobType:  contractJobs.CatalogImportPurgeRequestedType,
	}}
}

func NewInvoiceSupport(app *application.Application) api.InvoiceSupport {
	return application.NewInvoiceSupport(app)
}
