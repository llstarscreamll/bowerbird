package invoices

import (
	"context"
	"fmt"
	"net/http"

	catalogapi "github.com/atta/internal/catalog/api"
	inboxapi "github.com/atta/internal/inbox/api"
	invoicesEvents "github.com/atta/internal/invoices/adapters/events"
	invoicingLLM "github.com/atta/internal/invoices/adapters/extractors/llm"
	invoicingXML "github.com/atta/internal/invoices/adapters/extractors/xml"
	httpV1 "github.com/atta/internal/invoices/adapters/http/v1"
	invoicesJobs "github.com/atta/internal/invoices/adapters/jobs"
	invoiceLinking "github.com/atta/internal/invoices/adapters/linking"
	invoicingRepo "github.com/atta/internal/invoices/adapters/repository/postgres"
	invoicesapi "github.com/atta/internal/invoices/api"
	"github.com/atta/internal/invoices/application"
	"github.com/atta/internal/invoices/application/commands"
	"github.com/atta/internal/invoices/application/ports"
	"github.com/atta/internal/invoices/application/queries"
	partiesapi "github.com/atta/internal/parties/api"
	"github.com/atta/internal/platform/config"
	"github.com/atta/internal/platform/database"
	"github.com/atta/internal/platform/events"
	"github.com/atta/internal/platform/jobs"
	platformStorage "github.com/atta/internal/platform/storage"
	secretsapi "github.com/atta/internal/secrets/api"
)

func NewApplication(
	cfg config.Config,
	eventBus events.EventBus,
	jobQueue jobs.TaskQueue,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
	passwords secretsapi.DocumentPasswordResolver,
	catalog catalogapi.InvoiceSupport,
	parties partiesapi.IssuerPartyLookup,
	receivers ports.ReceiverDirectory,
	inboxBackfill inboxapi.InvoiceBackfillSource,
) *application.Application {
	if eventBus == nil {
		panic("event bus is required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}
	if fileStore == nil {
		panic("file store is required")
	}
	if registry == nil {
		panic("database registry is required")
	}
	if cfg.GeminiAPIKey == "" {
		panic("gemini api key is required")
	}
	if passwords == nil {
		panic("document password resolver is required")
	}
	if catalog == nil {
		panic("catalog invoice support is required")
	}
	if parties == nil {
		panic("issuer party lookup is required")
	}
	if receivers == nil {
		panic("receiver directory is required")
	}
	if inboxBackfill == nil {
		panic("inbox backfill source is required")
	}

	passwordResolver := newSecretsPasswordAdapter(passwords)
	partyResolver := invoiceLinking.NewPartyResolverAdapter(parties)
	catalogACL := invoiceLinking.NewCatalogACL(catalog)

	invoiceRepository := invoicingRepo.NewRepository(registry)
	xmlExtractor := invoicingXML.NewDianUBL21Parser()

	llmExtractor, err := invoicingLLM.NewGeminiExtractor(invoicingLLM.GeminiExtractorConfig{
		APIKey:   cfg.GeminiAPIKey,
		Model:    cfg.GeminiModel,
		Endpoint: cfg.GeminiEndpoint,
	})
	if err != nil {
		panic(fmt.Sprintf("new Gemini invoice extractor failed: %v", err))
	}

	createInvoice := commands.NewCreateInvoiceCommand(invoiceRepository, partyResolver, catalogACL, receivers)
	fromInbox := commands.NewCreateInvoicesFromInboxMessageCommand(jobQueue, receivers)
	backfill := commands.NewBackfillInboxInvoicesCommand(inboxBackfill, fromInbox, jobQueue, receivers)

	return &application.Application{
		Commands: application.Commands{
			CreateInvoicesFromInboxMessage:  fromInbox,
			QueueInvoiceExtractionFromFiles: commands.NewQueueInvoiceExtractionFromFilesCommand(jobQueue, receivers),
			ProcessInvoiceExtractionJob: commands.NewCreateInvoicesFromFilesCommand(
				fileStore,
				xmlExtractor,
				llmExtractor,
				invoiceRepository,
				passwordResolver,
				createInvoice,
			),
			CreateInvoice:           createInvoice,
			ApplyLineDecision:       commands.NewApplyLineDecisionCommand(invoiceRepository, catalogACL),
			DownloadInvoiceDocument: commands.NewDownloadInvoiceDocumentCommand(invoiceRepository, fileStore),
			BackfillInboxInvoices:   backfill,
		},
		Queries: application.Queries{
			GetInvoiceByID:  queries.NewGetInvoiceByIDQuery(invoiceRepository, catalogACL),
			ListInvoices:    queries.NewListInvoicesQuery(invoiceRepository),
			ListReviewQueue: queries.NewListReviewQueueQuery(invoiceRepository, catalogACL),
		},
		ItemLinks: application.NewItemLinkSupport(invoiceRepository),
	}
}

func NewItemLinkSupport(app *application.Application) invoicesapi.ItemLinkSupport {
	if app == nil {
		panic("invoicing application is required")
	}
	if app.ItemLinks == nil {
		panic("item link support is required")
	}
	return app.ItemLinks
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}

	if app == nil {
		panic("invoicing application is required")
	}

	controller := httpV1.NewController(app)
	handler := httpV1.NewRouter(controller)
	handler.Register(mux, cfg, authMiddleware)

	return handler
}

func RegisterEvents(app *application.Application) []events.IntegrationEventHandler {
	if app == nil {
		panic("invoicing application is required")
	}
	return []events.IntegrationEventHandler{
		invoicesEvents.NewInboxMessageReceivedSubscriber(app.Commands.CreateInvoicesFromInboxMessage),
		invoicesEvents.NewLegalEntityRegisteredSubscriber(app.Commands.BackfillInboxInvoices),
	}
}

func RegisterJobs(app *application.Application) []jobs.JobHandler {
	if app == nil {
		panic("invoicing application is required")
	}
	return []jobs.JobHandler{
		invoicesJobs.NewInvoiceExtractionRequestedProcessor(app.Commands.ProcessInvoiceExtractionJob),
		invoicesJobs.NewInvoiceInboxBackfillProcessor(app.Commands.BackfillInboxInvoices),
	}
}

type secretsPasswordAdapter struct {
	inner secretsapi.DocumentPasswordResolver
}

func newSecretsPasswordAdapter(resolver secretsapi.DocumentPasswordResolver) ports.DocumentPasswordResolver {
	return &secretsPasswordAdapter{inner: resolver}
}

func (a *secretsPasswordAdapter) ResolveCandidates(ctx context.Context) ([]ports.PasswordCandidate, error) {
	resolved, err := a.inner.ResolveCandidates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ports.PasswordCandidate, 0, len(resolved))
	for _, item := range resolved {
		out = append(out, ports.PasswordCandidate{SecretID: item.ID, Value: item.Value})
	}
	return out, nil
}

func (a *secretsPasswordAdapter) MarkUsed(ctx context.Context, secretID string) error {
	return a.inner.MarkUsed(ctx, secretID)
}
