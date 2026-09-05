package invoices

import (
	"context"
	"fmt"
	"net/http"

	catalogapi "github.com/bowerbird/internal/catalog/api"
	invoicesEvents "github.com/bowerbird/internal/invoices/adapters/events"
	invoicingLLM "github.com/bowerbird/internal/invoices/adapters/extractors/llm"
	invoicingXML "github.com/bowerbird/internal/invoices/adapters/extractors/xml"
	httpV1 "github.com/bowerbird/internal/invoices/adapters/http/v1"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	invoiceLinking "github.com/bowerbird/internal/invoices/adapters/linking"
	invoicingRepo "github.com/bowerbird/internal/invoices/adapters/repository/postgres"
	"github.com/bowerbird/internal/invoices/application"
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/application/queries"
	partiesapi "github.com/bowerbird/internal/parties/api"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	secretsapi "github.com/bowerbird/internal/secrets/api"
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

	createInvoice := commands.NewCreateInvoiceCommand(invoiceRepository, partyResolver, catalogACL)

	return &application.Application{
		Commands: application.Commands{
			CreateInvoicesFromInboxMessage:  commands.NewCreateInvoicesFromInboxMessageCommand(jobQueue),
			QueueInvoiceExtractionFromFiles: commands.NewQueueInvoiceExtractionFromFilesCommand(jobQueue),
			ProcessInvoiceExtractionJob: commands.NewCreateInvoicesFromFilesCommand(
				fileStore,
				xmlExtractor,
				llmExtractor,
				invoiceRepository,
				passwordResolver,
				createInvoice,
			),
			CreateInvoice:     createInvoice,
			ApplyLineDecision: commands.NewApplyLineDecisionCommand(invoiceRepository, catalogACL),
		},
		Queries: application.Queries{
			GetInvoiceByID:  queries.NewGetInvoiceByIDQuery(invoiceRepository, catalogACL),
			ListInvoices:    queries.NewListInvoicesQuery(invoiceRepository),
			ListReviewQueue: queries.NewListReviewQueueQuery(invoiceRepository, catalogACL),
		},
	}
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
	}
}

func RegisterJobs(app *application.Application) []jobs.JobHandler {
	if app == nil {
		panic("invoicing application is required")
	}
	return []jobs.JobHandler{
		invoicesJobs.NewInvoiceExtractionRequestedProcessor(app.Commands.ProcessInvoiceExtractionJob),
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
