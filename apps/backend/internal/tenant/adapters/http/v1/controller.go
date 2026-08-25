package v1

import (
	"encoding/json"
	"net/http"

	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
	"github.com/bowerbird/internal/tenant/application"
)

type Controller struct {
	createUseCase *application.CreateTenantUseCase
	getUseCase    *application.GetTenantUseCase
}

func NewController(createUseCase *application.CreateTenantUseCase, getUseCase *application.GetTenantUseCase) *Controller {
	if createUseCase == nil {
		panic("tenant create use case is required")
	}

	if getUseCase == nil {
		panic("tenant get use case is required")
	}

	return &Controller{
		createUseCase: createUseCase,
		getUseCase:    getUseCase,
	}
}

func (c *Controller) CreateTenant(w http.ResponseWriter, r *http.Request) error {
	var req createTenantRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	if err := req.Validate(); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	cmd := application.CreateTenantCommand{
		Name:           req.Name,
		Slug:           req.Slug,
		OwnerID:        claims.UserID,
		OwnerEmail:     claims.Email,
		OwnerFirstName: claims.FirstName,
		OwnerLastName:  claims.LastName,
		OwnerAvatarURL: claims.PictureURL,
	}

	org, err := c.createUseCase.Execute(r.Context(), cmd)
	if err != nil {
		if err == application.ErrSlugAlreadyExists {
			return appErrors.Wrap(err, appErrors.CodeConflict, "slug already exists")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to create tenant")
	}

	resp := newTenantResponse(org)

	return api.Success(w, http.StatusCreated, resp)
}

func (c *Controller) GetTenant(w http.ResponseWriter, r *http.Request) error {
	tenantID := r.PathValue("id")
	if tenantID == "" {
		return appErrors.New(appErrors.CodeValidation, "id is required")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	org, err := c.getUseCase.Execute(r.Context(), tenantID, claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeNotFound, "tenant not found")
	}

	resp := newTenantResponse(org)

	return api.Success(w, http.StatusOK, resp)
}
