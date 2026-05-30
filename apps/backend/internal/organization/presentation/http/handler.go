package http

import (
	"encoding/json"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	appErrors "github.com/money-path/bowerbird/apps/backend/internal/platform/errors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	createUseCase *application.CreateOrganizationUseCase
	getUseCase    *application.GetOrganizationUseCase
}

func NewHandler(createUseCase *application.CreateOrganizationUseCase, getUseCase *application.GetOrganizationUseCase) *Handler {
	return &Handler{
		createUseCase: createUseCase,
		getUseCase:    getUseCase,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("POST /api/v1/organizations", authMiddleware(api.Wrap(h.CreateOrganization, isDev)))
	mux.Handle("GET /api/v1/organizations/{id}", authMiddleware(api.Wrap(h.GetOrganization, isDev)))
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreateOrganizationResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	MembersCount    int    `json:"members_count,omitempty"`
	CurrentUserRole string `json:"current_user_role,omitempty"`
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) error {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	org, err := h.createUseCase.Execute(r.Context(), cmd)
	if err != nil {
		if err == application.ErrSlugAlreadyExists {
			return appErrors.Wrap(err, appErrors.CodeConflict, "slug already exists")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to create organization")
	}

	resp := CreateOrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    org.Status,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return api.Success(w, http.StatusCreated, resp)
}

func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return appErrors.New(appErrors.CodeValidation, "id is required")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	org, err := h.getUseCase.Execute(r.Context(), id, claims.UserID)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeNotFound, "organization not found")
	}

	resp := CreateOrganizationResponse{
		ID:              org.ID,
		Name:            org.Name,
		Slug:            org.Slug,
		Status:          org.Status,
		CreatedAt:       org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		MembersCount:    org.MembersCount,
		CurrentUserRole: org.CurrentUserRole,
	}

	return api.Success(w, http.StatusOK, resp)
}
