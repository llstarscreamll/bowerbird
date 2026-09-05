package identity

import (
	"context"
	"net/http"
	"strings"

	httpV1 "github.com/bowerbird/internal/identity/adapters/http/v1"
	identitypostgres "github.com/bowerbird/internal/identity/adapters/repository/postgres"
	"github.com/bowerbird/internal/identity/api"
	"github.com/bowerbird/internal/identity/application"
	"github.com/bowerbird/internal/identity/domain"
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

	identityRepo := identitypostgres.NewPostgresRepository(controlDB, tenantRegistry)
	refreshStore := identitypostgres.NewRefreshTokenRepository(controlDB)

	return application.NewApplication(identityRepo, tokenGen, refreshStore, cfg.AppEnv, cfg.PlatformOperatorEmails)
}

func NewHTTPHandler(mux *http.ServeMux, app *application.Application, controlDB *pgxpool.Pool, tenantRegistry *database.Registry, authMiddleware func(http.Handler) http.Handler, cfg config.Config) *httpV1.AuthHandler {
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

	handler := httpV1.NewAuthHandler(
		app.Commands.Auth,
		app.Identity,
		googleConfig,
		microsoftConfig,
		strings.TrimRight(cfg.FrontendURL, "/"),
		cfg.JWT.RefreshTTL,
	)
	handler.Register(mux, authMiddleware, cfg)

	return handler
}

func NewOperatorDirectory(app *application.Application) api.OperatorDirectory {
	if app == nil {
		panic("identity application is required")
	}
	return app
}

type UserStore interface {
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
}

func NewUserStore(controlDB *pgxpool.Pool, tenantRegistry *database.Registry) UserStore {
	if controlDB == nil {
		panic("control plane db pool is required")
	}
	if tenantRegistry == nil {
		panic("tenant registry is required")
	}
	return identitypostgres.NewPostgresRepository(controlDB, tenantRegistry)
}

var _ api.OperatorDirectory = (*application.Application)(nil)
