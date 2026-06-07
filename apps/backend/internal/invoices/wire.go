package invoices

import (
	"fmt"
	"net/http"

	invoicingLLM "github.com/bowerbird/internal/invoices/adapters/extractors/llm"
	invoicingXML "github.com/bowerbird/internal/invoices/adapters/extractors/xml"
	httpV1 "github.com/bowerbird/internal/invoices/adapters/http/v1"
	invoicingRepo "github.com/bowerbird/internal/invoices/adapters/repository/postgres"
	"github.com/bowerbird/internal/invoices/application"
	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func NewApplication(
	cfg config.Config,
	eventBus events.EventBus,
	jobQueue jobs.Queue,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
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
			ProcessInvoiceExtractionJob: commands.NewProcessInvoiceExtractionJobCommand(
				fileStore,
				xmlExtractor,
				llmExtractor,
				invoiceRepository,
			),
			CreateInvoice: commands.NewCreateInvoiceCommand(invoiceRepository),
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
