package connections

import (
	"net/http"
	"strings"

	eventsadapter "github.com/bowerbird/internal/connections/adapters/events"
	httpV1 "github.com/bowerbird/internal/connections/adapters/http/v1"
	repositorypostgres "github.com/bowerbird/internal/connections/adapters/repository/postgres"
	"github.com/bowerbird/internal/connections/api"
	"github.com/bowerbird/internal/connections/application"
	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

func NewApplication(registry *database.Registry, cipher application.CredentialsCipher) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	if cipher == nil {
		panic("credentials cipher is required")
	}

	connectionsRepo := repositorypostgres.NewPostgresRepository(registry)
	credentialsService := application.NewCredentialsService(cipher)

	return application.NewApplication(connectionsRepo, credentialsService)
}

func NewInternalService(app *application.Application) api.InternalService {
	return application.NewInternalService(app)
}

func NewHTTPHandler(mux *http.ServeMux, cfg config.Config, registry *database.Registry, cipher application.CredentialsCipher, tokenValidator httpV1.TokenValidator, stateProtector httpV1.StateProtector, eventBus events.EventBus, authMiddleware func(http.Handler) http.Handler, features entitlementsapi.Features) *httpV1.Router {
	if mux == nil {
		panic("http mux is required")
	}
	if registry == nil {
		panic("database registry is required")
	}
	if tokenValidator == nil {
		panic("token validator is required")
	}
	if eventBus == nil {
		panic("event bus is required")
	}
	if features == nil {
		panic("feature checker is required")
	}

	repo := repositorypostgres.NewPostgresRepository(registry)
	credentialsService := application.NewCredentialsService(cipher)

	var googleConfig *oauth2.Config
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/connections/google/callback",
			Scopes:       []string{"email", "https://www.googleapis.com/auth/gmail.modify", "https://www.googleapis.com/auth/gmail.send"},
			Endpoint:     google.Endpoint,
		}
	}

	var microsoftConfig *oauth2.Config
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		microsoftConfig = &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/connections/microsoft/callback",
			Scopes:       []string{"offline_access", "User.Read", "Mail.ReadWrite", "Mail.Send"},
			Endpoint:     microsoft.AzureADEndpoint("common"),
		}
	}

	controller := httpV1.NewController(
		repo,
		credentialsService,
		googleConfig,
		microsoftConfig,
		tokenValidator,
		stateProtector,
		eventsadapter.NewPublisher(eventBus),
		strings.TrimRight(cfg.FrontendURL, "/"),
		features,
	)
	router := httpV1.NewRouter(controller)
	router.Register(mux, cfg, authMiddleware)

	return router
}
