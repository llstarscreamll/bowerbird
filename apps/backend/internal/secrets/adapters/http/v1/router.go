package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	rbacapi "github.com/bowerbird/internal/rbac/api"
	"github.com/bowerbird/internal/secrets/application"
	"github.com/bowerbird/internal/secrets/application/commands"
	"github.com/bowerbird/internal/secrets/domain"
)

type Controller struct {
	app  *application.Application
	rbac rbacapi.Authorizer
}

func NewController(app *application.Application, rbac rbacapi.Authorizer) *Controller {
	if app == nil {
		panic("secrets application is required")
	}
	if rbac == nil {
		panic("rbac service is required")
	}
	return &Controller{app: app, rbac: rbac}
}

type secretAttributes struct {
	Purpose     string  `json:"purpose"`
	Kind        string  `json:"kind"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Version     int     `json:"version"`
	HasValue    bool    `json:"has_value"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type secretResource struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Attributes secretAttributes `json:"attributes"`
}

type secretCollectionResponse struct {
	Data []secretResource `json:"data"`
}

type secretDocumentResponse struct {
	Data secretResource `json:"data"`
}

type createSecretRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Purpose     string `json:"purpose"`
			Kind        string `json:"kind"`
			Label       string `json:"label"`
			Description string `json:"description"`
			Value       string `json:"value"`
		} `json:"attributes"`
	} `json:"data"`
}

type updateSecretRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Label       *string `json:"label"`
			Description *string `json:"description"`
			Value       *string `json:"value"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Controller) ListSecrets(w http.ResponseWriter, r *http.Request) error {
	if err := c.rbac.RequirePermission(r.Context(), rbacapi.PermissionSecretsRead); err != nil {
		return err
	}

	purpose := r.URL.Query().Get("purpose")
	secrets, err := c.app.Queries.ListSecrets.Execute(r.Context(), purpose)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list secrets")
	}

	data := make([]secretResource, 0, len(secrets))
	for _, secret := range secrets {
		data = append(data, toSecretResource(secret))
	}
	return api.Success(w, http.StatusOK, secretCollectionResponse{Data: data})
}

func (c *Controller) GetSecret(w http.ResponseWriter, r *http.Request) error {
	if err := c.rbac.RequirePermission(r.Context(), rbacapi.PermissionSecretsRead); err != nil {
		return err
	}

	secret, err := c.app.Queries.GetSecretByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, secretDocumentResponse{Data: toSecretResource(*secret)})
}

func (c *Controller) CreateSecret(w http.ResponseWriter, r *http.Request) error {
	if err := c.rbac.RequirePermission(r.Context(), rbacapi.PermissionSecretsWrite); err != nil {
		return err
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	var req createSecretRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	secret, err := c.app.Commands.CreateSecret.Execute(r.Context(), commands.CreateSecretInput{
		Purpose:     req.Data.Attributes.Purpose,
		Kind:        req.Data.Attributes.Kind,
		Label:       req.Data.Attributes.Label,
		Description: req.Data.Attributes.Description,
		Value:       req.Data.Attributes.Value,
		ActorUserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return api.Success(w, http.StatusCreated, secretDocumentResponse{Data: toSecretResource(*secret)})
}

func (c *Controller) UpdateSecret(w http.ResponseWriter, r *http.Request) error {
	if err := c.rbac.RequirePermission(r.Context(), rbacapi.PermissionSecretsWrite); err != nil {
		return err
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	var req updateSecretRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	secret, err := c.app.Commands.UpdateSecret.Execute(r.Context(), commands.UpdateSecretInput{
		ID:          r.PathValue("id"),
		Label:       req.Data.Attributes.Label,
		Description: req.Data.Attributes.Description,
		Value:       req.Data.Attributes.Value,
		ActorUserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return api.Success(w, http.StatusOK, secretDocumentResponse{Data: toSecretResource(*secret)})
}

func (c *Controller) DeleteSecret(w http.ResponseWriter, r *http.Request) error {
	if err := c.rbac.RequirePermission(r.Context(), rbacapi.PermissionSecretsDelete); err != nil {
		return err
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	if err := c.app.Commands.DeleteSecret.Execute(r.Context(), r.PathValue("id"), claims.UserID); err != nil {
		return err
	}
	return api.Success(w, http.StatusNoContent, nil)
}

func toSecretResource(secret domain.Secret) secretResource {
	var lastUsed *string
	if secret.LastUsedAt != nil {
		formatted := secret.LastUsedAt.UTC().Format(time.RFC3339)
		lastUsed = &formatted
	}
	return secretResource{
		Type: "secrets",
		ID:   secret.ID,
		Attributes: secretAttributes{
			Purpose:     secret.Purpose,
			Kind:        secret.Kind,
			Label:       secret.Label,
			Description: secret.Description,
			Version:     secret.Version,
			HasValue:    true,
			LastUsedAt:  lastUsed,
			CreatedAt:   secret.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:   secret.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}
}

type Router struct {
	controller *Controller
}

func NewRouter(controller *Controller) *Router {
	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/secrets", authMiddleware(api.Wrap(h.controller.ListSecrets, cfg)))
	mux.Handle("GET /api/v1/secrets/{id}", authMiddleware(api.Wrap(h.controller.GetSecret, cfg)))
	mux.Handle("POST /api/v1/secrets", authMiddleware(api.Wrap(h.controller.CreateSecret, cfg)))
	mux.Handle("PUT /api/v1/secrets/{id}", authMiddleware(api.Wrap(h.controller.UpdateSecret, cfg)))
	mux.Handle("DELETE /api/v1/secrets/{id}", authMiddleware(api.Wrap(h.controller.DeleteSecret, cfg)))
}
