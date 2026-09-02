package invoices

import (
	"context"
	"fmt"
	"net/http"

	invoicesEvents "github.com/bowerbird/internal/invoices/adapters/events"
	invoicingLLM "github.com/bowerbird/internal/invoices/adapters/extractors/llm"
	invoicingXML "github.com/bowerbird/internal/invoices/adapters/extractors/xml"
	httpV1 "github.com/bowerbird/internal/invoices/adapters/http/v1"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	invoicingRepo "github.com/bowerbird/internal/invoices/adapters/repository/postgres"
	"github.com/bowerbird/internal/invoices/application"
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/application/queries"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	secretsModule "github.com/bowerbird/internal/secrets"
)

func NewApplication(
	cfg config.Config,
	eventBus events.EventBus,
	jobQueue jobs.TaskQueue,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
	passwordResolver ports.DocumentPasswordResolver,
	partyResolver ports.IssuerPartyResolver,
	lineResolver ports.CatalogLineResolver,
	catalogService ports.CatalogService,
	catalogMatching ports.CatalogMatchingPort,
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

	createInvoice := commands.NewCreateInvoiceCommand(invoiceRepository, partyResolver, lineResolver)

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
			ApplyLineDecision: commands.NewApplyLineDecisionCommand(invoiceRepository, catalogMatching),
		},
		Queries: application.Queries{
			GetInvoiceByID:  queries.NewGetInvoiceByIDQuery(invoiceRepository, catalogService),
			ListInvoices:    queries.NewListInvoicesQuery(invoiceRepository),
			ListReviewQueue: queries.NewListReviewQueueQuery(invoiceRepository, catalogService),
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

// RegisterMessaging wires invoices integration event and job handlers for the messaging composition root.
func RegisterMessaging(app *application.Application) ([]events.IntegrationEventHandler, []jobs.JobHandler) {
	if app == nil {
		panic("invoicing application is required")
	}
	return []events.IntegrationEventHandler{
			invoicesEvents.NewInboxMessageReceivedSubscriber(app.Commands.CreateInvoicesFromInboxMessage),
		}, []jobs.JobHandler{
			invoicesJobs.NewInvoiceExtractionRequestedProcessor(app.Commands.ProcessInvoiceExtractionJob),
		}
}

type secretsPasswordAdapter struct {
	inner *secretsModule.DocumentPasswordResolver
}

func NewSecretsPasswordAdapter(resolver *secretsModule.DocumentPasswordResolver) ports.DocumentPasswordResolver {
	if resolver == nil {
		return nil
	}
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
