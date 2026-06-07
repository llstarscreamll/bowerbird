package v1

import (
	"encoding/json"
	"net/http"

	"github.com/bowerbird/internal/organization/application"
	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	createUseCase *application.CreateOrganizationUseCase
	getUseCase    *application.GetOrganizationUseCase
}

func NewController(createUseCase *application.CreateOrganizationUseCase, getUseCase *application.GetOrganizationUseCase) *Controller {
	if createUseCase == nil {
		panic("organization create use case is required")
	}

	if getUseCase == nil {
		panic("organization get use case is required")
	}

	return &Controller{
		createUseCase: createUseCase,
		getUseCase:    getUseCase,
	}
}

func (c *Controller) CreateOrganization(w http.ResponseWriter, r *http.Request) error {
	var req createOrganizationRequest
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

	cmd := application.CreateOrganizationCommand{
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
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to create organization")
	}

	resp := newOrganizationResponse(org)

	return api.Success(w, http.StatusCreated, resp)
}

func (c *Controller) GetOrganization(w http.ResponseWriter, r *http.Request) error {
	organizationID := r.PathValue("id")
	if organizationID == "" {
		return appErrors.New(appErrors.CodeValidation, "id is required")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	org, err := c.getUseCase.Execute(r.Context(), organizationID, claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeNotFound, "organization not found")
	}

	resp := newOrganizationResponse(org)

	return api.Success(w, http.StatusOK, resp)
}
