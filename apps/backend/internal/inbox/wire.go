package inbox

import (
	"net/http"

	connectionsApp "github.com/bowerbird/internal/connections/application"
	httpV1 "github.com/bowerbird/internal/inbox/adapters/http/v1"
	inboxJobs "github.com/bowerbird/internal/inbox/adapters/jobs"
	"github.com/bowerbird/internal/inbox/adapters/provider"
	"github.com/bowerbird/internal/inbox/adapters/provider/gmail"
	"github.com/bowerbird/internal/inbox/adapters/provider/microsoft"
	inboxRepo "github.com/bowerbird/internal/inbox/adapters/repository/postgres"
	"github.com/bowerbird/internal/inbox/application"
	"github.com/bowerbird/internal/inbox/application/commands"
	inboxPorts "github.com/bowerbird/internal/inbox/application/ports"
	"github.com/bowerbird/internal/inbox/application/queries"
	eventsV1 "github.com/bowerbird/internal/inbox/presentation/events"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func NewApplication(
	cfg config.Config,
	connectionsService connectionsApp.InternalService,
	eventBus events.EventBus,
	fileStore platformStorage.FileStore,
	registry *database.Registry,
	jobQueue jobs.Queue,
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
	var modifyMessageCommand *commands.ModifyMessageCommand
	var sendMessageCommand *commands.SendMessageCommand
	var downloadAttachmentCommand *commands.DownloadAttachmentCommand

	hasGmail := cfg.GoogleClientID != "" && cfg.GoogleClientSecret != ""
	hasMicrosoft := cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != ""
	if hasGmail || hasMicrosoft {
		if eventBus == nil {
			panic("event bus is required for inbox sync")
		}
		if fileStore == nil {
			panic("file store is required for inbox sync")
		}

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

		syncAccountCommand = commands.NewSyncAccountCommand(
			inboxRepository,
			inboxRepository,
			connectionsService,
			providerFactory,
			eventBus,
			fileStore,
		)
		modifyMessageCommand = commands.NewModifyMessageCommand(inboxRepository, connectionsService, providerFactory)
		sendMessageCommand = commands.NewSendMessageCommand(inboxRepository, connectionsService, providerFactory)
		downloadAttachmentCommand = commands.NewDownloadAttachmentCommand(inboxRepository, fileStore)

		var dispatcher commands.SyncAccountJobDispatcher
		if jobQueue != nil {
			dispatcher = commands.NewSQSSyncAccountJobDispatcher(jobQueue)
		} else {
			dispatcher = commands.NewInlineSyncAccountJobDispatcher(syncAccountCommand)
		}
		syncAllAccountsCommand = commands.NewSyncAllAccountsCommand(connectionsService, dispatcher)
	}

	return &application.Application{
		Commands: application.Commands{
			SyncAccount:        syncAccountCommand,
			SyncAllAccounts:    syncAllAccountsCommand,
			ModifyMessage:      modifyMessageCommand,
			SendMessage:        sendMessageCommand,
			DownloadAttachment: downloadAttachmentCommand,
		},
		Queries: application.Queries{
			ListAccountHealth: queries.NewListAccountHealthQuery(inboxRepository, connectionsService),
			ListMessages:      queries.NewListMessagesQuery(inboxRepository),
			GetMessage:        queries.NewGetMessageQuery(inboxRepository),
		},
	}
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, authMiddleware func(http.Handler) http.Handler, cfg config.Config, features inboxPorts.FeatureChecker) *httpV1.Router {
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
		app.Commands.ModifyMessage,
		app.Commands.SendMessage,
		app.Commands.DownloadAttachment,
		features,
	)
	handler := httpV1.NewRouter(controller)
	handler.Register(mux, cfg, authMiddleware)
	return handler
}

func NewConnectionAddedSubscriber(app *application.Application, features inboxPorts.FeatureChecker) *eventsV1.ConnectionAddedSubscriber {
	if app == nil {
		panic("inbox application is required")
	}
	return eventsV1.NewConnectionAddedSubscriber(app.Commands.SyncAccount, features)
}

func NewSyncAccountProcessor(app *application.Application, features inboxPorts.FeatureChecker) *inboxJobs.ProcessInboxSyncAccount {
	if app == nil {
		panic("inbox application is required")
	}
	return inboxJobs.NewProcessInboxSyncAccount(app.Commands.SyncAccount, features)
}
