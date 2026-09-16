package inbox

import (
	"context"
	"net/http"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	eventsV1 "github.com/bowerbird/internal/inbox/adapters/events"
	httpV1 "github.com/bowerbird/internal/inbox/adapters/http/v1"
	inboxJobs "github.com/bowerbird/internal/inbox/adapters/jobs"
	"github.com/bowerbird/internal/inbox/adapters/provider"
	"github.com/bowerbird/internal/inbox/adapters/provider/gmail"
	"github.com/bowerbird/internal/inbox/adapters/provider/microsoft"
	inboxRepo "github.com/bowerbird/internal/inbox/adapters/repository/postgres"
	inboxapi "github.com/bowerbird/internal/inbox/api"
	"github.com/bowerbird/internal/inbox/application"
	"github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/application/ports"
	"github.com/bowerbird/internal/inbox/application/queries"
	inboxContracts "github.com/bowerbird/internal/inbox/contracts/jobs"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/scheduler"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func NewApplication(
	cfg config.Config,
	connectionsService connectionsapi.InternalService,
	eventBus events.EventBus,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
	jobQueue jobs.TaskQueue,
) *application.Application {
	if connectionsService == nil {
		panic("connections internal service is required")
	}
	if registry == nil {
		panic("database registry is required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}

	inboxRepository := inboxRepo.NewPostgresRepository(registry)

	providerFactory := provider.NewDefaultFactoryWithConfig(provider.DefaultFactoryConfig{
		Gmail: gmail.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		},
		Microsoft: microsoft.OAuthConfig{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
		},
	})

	syncAllAccountsCommand := commands.NewSyncAllAccountsCommand(
		connectionsService,
		commands.NewOutboxSyncAccountJobDispatcher(jobQueue),
	)
	modifyMessageCommand := commands.NewModifyMessageCommand(inboxRepository, connectionsService, providerFactory)
	sendMessageCommand := commands.NewSendMessageCommand(inboxRepository, connectionsService, providerFactory)

	var downloadAttachmentCommand *commands.DownloadAttachmentCommand
	if fileStore != nil {
		downloadAttachmentCommand = commands.NewDownloadAttachmentCommand(inboxRepository, fileStore)
	}

	var syncAccountCommand *commands.SyncAccountCommand
	if mailSyncEnabled(cfg) {
		if eventBus == nil {
			panic("event bus is required for inbox sync")
		}
		if fileStore == nil {
			panic("file store is required for inbox sync")
		}

		syncAccountCommand = commands.NewSyncAccountCommand(
			inboxRepository,
			inboxRepository,
			connectionsService,
			providerFactory,
			eventBus,
			fileStore,
			database.NewRegistryUnitOfWork(registry),
		)
	}

	return &application.Application{
		Commands: application.Commands{
			SyncAccount:        syncAccountCommand,
			SyncAllAccounts:    syncAllAccountsCommand,
			ModifyMessage:      modifyMessageCommand,
			SendMessage:        sendMessageCommand,
			DownloadAttachment: downloadAttachmentCommand,
			HydrateMessage:     commands.NewHydrateMessageCommand(syncAccountCommand),
		},
		Queries: application.Queries{
			ListAccountHealth:        queries.NewListAccountHealthQuery(inboxRepository, connectionsService),
			ListMessages:             queries.NewListMessagesQuery(inboxRepository),
			GetMessage:               queries.NewGetMessageQuery(inboxRepository),
			ListExtractionCandidates: queries.NewListExtractionCandidatesQuery(inboxRepository),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config, features entitlementsapi.Features) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("inbox application is required")
	}
	if features == nil {
		panic("feature checker is required")
	}

	controller := httpV1.NewController(
		app.Queries.ListAccountHealth,
		app.Queries.ListMessages,
		app.Queries.GetMessage,
		app.Commands.SyncAllAccounts,
		app.Commands.ModifyMessage,
		app.Commands.SendMessage,
		app.Commands.DownloadAttachment,
		app.Commands.HydrateMessage,
		features,
	)
	handler := httpV1.NewRouter(controller)
	handler.Register(mux, cfg, authMiddleware)
	return handler
}

func RegisterEvents(
	features entitlementsapi.Features,
	taskQueue jobs.TaskQueue,
) []events.IntegrationEventHandler {
	if features == nil {
		panic("feature checker is required")
	}
	if taskQueue == nil {
		panic("job queue is required")
	}

	return []events.IntegrationEventHandler{
		eventsV1.NewConnectionAddedSubscriber(
			commands.NewOutboxSyncAccountJobDispatcher(taskQueue),
			features,
		),
	}
}

func RegisterJobs(
	app *application.Application,
	features entitlementsapi.Features,
	tenants ports.ActiveTenantLister,
) []jobs.JobHandler {
	if app == nil {
		panic("inbox application is required")
	}
	if features == nil {
		panic("feature checker is required")
	}

	var handlers []jobs.JobHandler
	if app.Commands.SyncAccount != nil {
		handlers = append(handlers, inboxJobs.NewProcessInboxSyncAccount(app.Commands.SyncAccount, features))
	}
	if app.Commands.SyncAllAccounts != nil {
		if tenants == nil {
			panic("tenant lister is required")
		}
		handlers = append(handlers, inboxJobs.NewProcessInboxSyncAllAccounts(
			commands.NewPlatformSyncAllAccountsCommand(app.Commands.SyncAllAccounts, features, tenants),
		))
	}
	return handlers
}

func RegisterSchedules(cfg config.Config) []scheduler.Rule {
	if !mailSyncEnabled(cfg) {
		return nil
	}
	return []scheduler.Rule{{
		Name:     "inbox-sync-all",
		Schedule: "rate(5 minutes)",
		JobType:  inboxContracts.InboxSyncAllAccountsType,
	}}
}

func NewInvoiceBackfillSource(app *application.Application) inboxapi.InvoiceBackfillSource {
	if app == nil {
		panic("inbox application is required")
	}
	if app.Queries.ListExtractionCandidates == nil {
		panic("list extraction candidates query is required")
	}
	return extractionBackfillSource{query: app.Queries.ListExtractionCandidates}
}

type extractionBackfillSource struct {
	query *queries.ListExtractionCandidatesQuery
}

func (s extractionBackfillSource) ListExtractionCandidates(ctx context.Context, cursor string, limit int) (inboxapi.ExtractionCandidatePage, error) {
	return s.query.Execute(ctx, cursor, limit)
}

func mailSyncEnabled(cfg config.Config) bool {
	hasGmail := cfg.GoogleClientID != "" && cfg.GoogleClientSecret != ""
	hasMicrosoft := cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != ""
	return hasGmail || hasMicrosoft
}
