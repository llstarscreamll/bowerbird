package http

import (
	"encoding/json"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/organization/application"
)

type Handler struct {
	createUseCase *application.CreateOrganizationUseCase
}

func NewHandler(createUseCase *application.CreateOrganizationUseCase) *Handler {
	return &Handler{
		createUseCase: createUseCase,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/organizations", h.CreateOrganization)
}

type CreateOrganizationRequest struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	OwnerID string `json:"owner_id"` // Simulation of authenticated user ID for now
}

type CreateOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cmd := application.CreateOrganizationCommand{
		Name:    req.Name,
		Slug:    req.Slug,
		OwnerID: req.OwnerID,
	}

	org, err := h.createUseCase.Execute(r.Context(), cmd)
	if err != nil {
		// Basic error mapping
		if err == application.ErrSlugAlreadyExists {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CreateOrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    org.Status,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
