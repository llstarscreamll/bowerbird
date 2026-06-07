package identity

import (
	"net/http"
	"strings"

	"github.com/bowerbird/internal/identity/application"
	identityinfra "github.com/bowerbird/internal/identity/infrastructure"
	identityhttp "github.com/bowerbird/internal/identity/presentation/http"
	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

func NewApplication(cfg config.Config, controlDB *pgxpool.Pool, tenantRegistry *database.Registry, tokenGen *auth.TokenGenerator) *application.Application {
	if controlDB == nil {
		panic("control plane db pool is required")
	}
	if tenantRegistry == nil {
		panic("tenant registry is required")
	}
	if tokenGen == nil {
		panic("token generator is required")
	}

	identityRepo := identityinfra.NewPostgresRepository(controlDB, tenantRegistry)

	return application.NewApplication(identityRepo, tokenGen, cfg.AppEnv)
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, controlDB *pgxpool.Pool, tenantRegistry *database.Registry, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *identityhttp.AuthHandler {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("identity application is required")
	}
	if controlDB == nil {
		panic("control plane db pool is required")
	}
	if tenantRegistry == nil {
		panic("tenant registry is required")
	}

	repo := identityinfra.NewPostgresRepository(controlDB, tenantRegistry)

	var googleConfig *oauth2.Config
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/auth/google/callback",
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	var microsoftConfig *oauth2.Config
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		microsoftConfig = &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/auth/microsoft/callback",
			Scopes:       []string{"User.Read"},
			Endpoint:     microsoft.AzureADEndpoint("common"),
		}
	}

	handler := identityhttp.NewAuthHandler(
		app.Commands.Auth,
		application.NewIdentityService(repo),
		googleConfig,
		microsoftConfig,
		strings.TrimRight(cfg.FrontendURL, "/"),
	)
	handler.Register(mux, authMiddleware, cfg)

	return handler
}
