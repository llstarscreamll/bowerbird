package connections

import (
	"context"
	"net/http"
	"strings"

	eventsadapter "github.com/bowerbird/internal/connections/adapters/events"
	httpV1 "github.com/bowerbird/internal/connections/adapters/http/v1"
	repositorypostgres "github.com/bowerbird/internal/connections/adapters/repository/postgres"
	"github.com/bowerbird/internal/connections/application"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type internalService struct {
	app *application.Application
}

func NewApplication(registry *database.Registry, cipher application.CredentialsCipher) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}

	connectionsRepo := repositorypostgres.NewPostgresRepository(registry)
	credentialsService := application.NewCredentialsService(cipher)

	return application.NewApplication(connectionsRepo, credentialsService)
}

func NewInternalService(app *application.Application) application.InternalService {
	if app == nil {
		panic("connections application is required")
	}

	return &internalService{app: app}
}

func (s *internalService) GetActiveConnections(ctx context.Context) ([]application.ConnectionInfo, error) {
	return s.app.Queries.GetActiveConnections.Execute(ctx)
}

func (s *internalService) DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error) {
	return s.app.Queries.DecryptCredentials.Execute(ctx, connectionID)
}

func (s *internalService) MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error {
	return s.app.Commands.MarkRequiresReconnect.Execute(ctx, connectionID, reason)
}

func (s *internalService) GetSharingPolicy(ctx context.Context, connectionID string) (string, error) {
	return s.app.Queries.GetSharingPolicy.Execute(ctx, connectionID)
}

func NewHTTPHandler(mux *http.ServeMux, cfg config.Config, registry *database.Registry, cipher application.CredentialsCipher, tokenValidator httpV1.TokenValidator, stateProtector httpV1.StateProtector, eventBus events.EventBus, authMiddleware func(http.Handler) http.Handler) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if registry == nil {
		panic("database registry is required")
	}
	if tokenValidator == nil {
		panic("token validator is required")
	}

	repo := repositorypostgres.NewPostgresRepository(registry)
	credentialsService := application.NewCredentialsService(cipher)

	var googleConfig *oauth2.Config
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/connections/google/callback",
			Scopes:       []string{"email", "https://www.googleapis.com/auth/gmail.modify"},
			Endpoint:     google.Endpoint,
		}
	}

	var eventPublisher httpV1.EventPublisher
	if eventBus != nil {
		eventPublisher = eventsadapter.NewPublisher(eventBus)
	}

	controller := httpV1.NewController(
		repo,
		credentialsService,
		googleConfig,
		tokenValidator,
		stateProtector,
		eventPublisher,
		strings.TrimRight(cfg.FrontendURL, "/"),
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}
