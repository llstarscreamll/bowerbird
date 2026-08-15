package invoices

import (
	"context"
	"fmt"
	"net/http"

	invoicingLLM "github.com/bowerbird/internal/invoices/adapters/extractors/llm"
	invoicingXML "github.com/bowerbird/internal/invoices/adapters/extractors/xml"
	httpV1 "github.com/bowerbird/internal/invoices/adapters/http/v1"
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
	jobQueue jobs.Queue,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
	passwordResolver ports.DocumentPasswordResolver,
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
			),
			CreateInvoice: commands.NewCreateInvoiceCommand(invoiceRepository),
		},
		Queries: application.Queries{
			GetInvoiceByID: queries.NewGetInvoiceByIDQuery(invoiceRepository),
			ListInvoices:   queries.NewListInvoicesQuery(invoiceRepository),
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
