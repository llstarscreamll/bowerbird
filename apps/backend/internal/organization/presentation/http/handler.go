package http

import (
	"encoding/json"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	createUseCase *application.CreateOrganizationUseCase
}

func NewHandler(createUseCase *application.CreateOrganizationUseCase) *Handler {
	return &Handler{
		createUseCase: createUseCase,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("POST /api/v1/organizations", authMiddleware(api.Wrap(h.CreateOrganization, isDev)))
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreateOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) error {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return apperrors.Wrap(err, apperrors.CodeValidation, "invalid request body")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return apperrors.New(apperrors.CodeUnauthorized, "unauthorized")
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
			return apperrors.Wrap(err, apperrors.CodeConflict, "slug already exists")
		}
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to create organization")
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
