package secrets

import (
	"context"
	"net/http"

	"github.com/atta/internal/platform/config"
	"github.com/atta/internal/platform/database"
	rbacapi "github.com/atta/internal/rbac/api"
	httpV1 "github.com/atta/internal/secrets/adapters/http/v1"
	secretsRepo "github.com/atta/internal/secrets/adapters/repository/postgres"
	"github.com/atta/internal/secrets/api"
	"github.com/atta/internal/secrets/application"
	"github.com/atta/internal/secrets/application/commands"
	"github.com/atta/internal/secrets/application/ports"
	"github.com/atta/internal/secrets/application/queries"
	"github.com/atta/internal/secrets/domain"
)

func NewApplication(registry *database.Registry, cipher ports.SecretCipher) *application.Application {
	if registry == nil {
		panic("database registry is required")
	}
	if cipher == nil {
		panic("secret cipher is required")
	}

	repo := secretsRepo.NewSecretRepository(registry)
	return &application.Application{
		Commands: application.Commands{
			CreateSecret:   commands.NewCreateSecretCommand(repo, cipher),
			UpdateSecret:   commands.NewUpdateSecretCommand(repo, cipher),
			DeleteSecret:   commands.NewDeleteSecretCommand(repo),
			MarkSecretUsed: commands.NewMarkSecretUsedCommand(repo),
		},
		Queries: application.Queries{
			ListSecrets:      queries.NewListSecretsQuery(repo),
			GetSecretByID:    queries.NewGetSecretByIDQuery(repo),
			ResolveByPurpose: queries.NewResolveByPurposeQuery(repo, cipher),
		},
	}
}

func NewHTTPHandler(
	mux *http.ServeMux,
	app *application.Application,
	rbac rbacapi.Authorizer,
	authMiddleware func(http.Handler) http.Handler,
	cfg config.Config,
) {
	if mux == nil {
		panic("http mux is required")
	}
	if app == nil {
		panic("secrets application is required")
	}
	controller := httpV1.NewController(app, rbac)
	httpV1.NewRouter(controller).Register(mux, cfg, authMiddleware)
}

type documentPasswordResolver struct {
	app *application.Application
}

func NewDocumentPasswordResolver(app *application.Application) api.DocumentPasswordResolver {
	if app == nil {
		panic("secrets application is required")
	}
	return &documentPasswordResolver{app: app}
}

func (r *documentPasswordResolver) ResolveCandidates(ctx context.Context) ([]api.PasswordCandidate, error) {
	resolved, err := r.app.Queries.ResolveByPurpose.Execute(ctx, domain.PurposeInvoicingDocumentPassword)
	if err != nil {
		return nil, err
	}
	out := make([]api.PasswordCandidate, 0, len(resolved))
	for _, item := range resolved {
		out = append(out, api.PasswordCandidate{ID: item.ID, Value: item.Value})
	}
	return out, nil
}

func (r *documentPasswordResolver) MarkUsed(ctx context.Context, secretID string) error {
	return r.app.Commands.MarkSecretUsed.Execute(ctx, secretID)
}
