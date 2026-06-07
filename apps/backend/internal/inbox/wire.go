package inbox

import (
	"net/http"

	connectionsApp "github.com/bowerbird/internal/connections/application"
	httpV1 "github.com/bowerbird/internal/inbox/adapters/http/v1"
	"github.com/bowerbird/internal/inbox/adapters/provider"
	"github.com/bowerbird/internal/inbox/adapters/provider/gmail"
	inboxRepo "github.com/bowerbird/internal/inbox/adapters/repository/postgres"
	"github.com/bowerbird/internal/inbox/application"
	"github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/application/queries"
	eventsV1 "github.com/bowerbird/internal/inbox/presentation/events"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func NewApplication(
	cfg config.Config,
	connectionsService connectionsApp.InternalService,
	eventBus events.EventBus,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
) *application.Application {
	if connectionsService == nil {
		panic("connections internal service is required")
	}

	if registry == nil {
		panic("database registry is required")
	}

	inboxRepository := inboxRepo.NewPostgresRepository(registry)

	var syncAccountCommand *commands.SyncAccountCommand
	var syncAllAccountsCommand *commands.SyncAllAccountsCommand

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		if eventBus == nil {
			panic("event bus is required for inbox sync")
		}

		if fileStore == nil {
			panic("file store is required for inbox sync")
		}

		providerFactory := provider.NewDefaultFactory(gmail.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})

		syncAccountCommand = commands.NewSyncAccountCommand(
			inboxRepository,
			inboxRepository,
			connectionsService,
			providerFactory,
			eventBus,
			fileStore,
		)

		syncAccountJobDispatcher := commands.NewInlineSyncAccountJobDispatcher(syncAccountCommand)
		syncAllAccountsCommand = commands.NewSyncAllAccountsCommand(connectionsService, syncAccountJobDispatcher)
	}

	return &application.Application{
		Commands: application.Commands{
			SyncAccount:     syncAccountCommand,
			SyncAllAccounts: syncAllAccountsCommand,
		},
		Queries: application.Queries{
			ListAccountHealth: queries.NewListAccountHealthQuery(inboxRepository, connectionsService),
			ListMessages:      queries.NewListMessagesQuery(inboxRepository),
			GetMessage:        queries.NewGetMessageQuery(inboxRepository),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}

	if app == nil {
		panic("inbox application is required")
	}

	controller := httpV1.NewController(
		app.Queries.ListAccountHealth,
		app.Queries.ListMessages,
		app.Queries.GetMessage,
		app.Commands.SyncAllAccounts,
	)
	handler := httpV1.NewRouter(controller)
	handler.Register(mux, cfg, authMiddleware)

	return handler
}

func NewConnectionAddedSubscriber(app *application.Application) *eventsV1.ConnectionAddedSubscriber {
	if app == nil {
		panic("inbox application is required")
	}

	return eventsV1.NewConnectionAddedSubscriber(app.Commands.SyncAccount)
}
