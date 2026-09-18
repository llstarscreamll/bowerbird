package catalog

import (
	"net/http"

	httpV1 "github.com/atta/internal/catalog/adapters/http/v1"
	"github.com/atta/internal/catalog/adapters/invoicelinks"
	catalogJobs "github.com/atta/internal/catalog/adapters/jobs"
	"github.com/atta/internal/catalog/adapters/matchers"
	catalogRepo "github.com/atta/internal/catalog/adapters/repository/postgres"
	"github.com/atta/internal/catalog/api"
	"github.com/atta/internal/catalog/application"
	"github.com/atta/internal/catalog/application/commands"
	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/application/queries"
	contractJobs "github.com/atta/internal/catalog/contracts/jobs"
	filesapi "github.com/atta/internal/files/api"
	invoicesapi "github.com/atta/internal/invoices/api"
	"github.com/atta/internal/platform/config"
	"github.com/atta/internal/platform/database"
	"github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/scheduler"
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
			AddItemAlias:                commands.NewAddItemAliasCommand(repo, repo),
			RemoveItemAlias:             commands.NewRemoveItemAliasCommand(repo, repo),
			RememberDecision:            commands.NewRememberDecisionCommand(repo, repo, repo),
			MergeItems:                  commands.NewMergeItemsCommand(repo, repo, repo, nil),
			MarkNotDuplicates:           commands.NewMarkNotDuplicatesCommand(repo, repo),
			QueueCatalogImport:          commands.NewQueueCatalogImportCommand(repo, objects, jobQueue),
			ProcessCatalogImport:        commands.NewProcessCatalogImportCommand(repo, repo, objects, jobQueue),
			CancelCatalogImport:         commands.NewCancelCatalogImportCommand(repo),
			PurgeStaleCatalogImports:    commands.NewPurgeStaleCatalogImportsCommand(repo),
		},
		Queries: application.Queries{
			GetItemByID:           queries.NewGetItemByIDQuery(repo, repo),
			GetItemNames:          queries.NewGetItemNamesQuery(repo),
			GetItemDisplays:       queries.NewGetItemDisplaysQuery(repo),
			ListItems:             queries.NewListItemsQuery(repo),
			ListDuplicateClusters: queries.NewListDuplicateClustersQuery(repo, repo, repo, repo, nil),
			GetImportByID:         queries.NewGetImportByIDQuery(repo),
			GetActiveImport:       queries.NewGetActiveImportQuery(repo),
			ListImports:           queries.NewListImportsQuery(repo),
			ListImportErrors:      queries.NewListImportErrorsQuery(repo),
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

func BindItemLinks(app *application.Application, links invoicesapi.ItemLinkSupport) {
	if app == nil {
		panic("catalog application is required")
	}
	app.BindItemLinks(invoicelinks.New(links))
}
