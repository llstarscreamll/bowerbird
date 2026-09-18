package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/atta/internal/legalentities/application"
	"github.com/atta/internal/legalentities/application/commands"
	"github.com/atta/internal/legalentities/domain"
	"github.com/atta/internal/platform/config"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/http/api"
)

type Controller struct {
	app *application.Application
}

func NewController(app *application.Application) *Controller {
	if app == nil {
		panic("legalentities application is required")
	}
	return &Controller{app: app}
}

type legalEntityAttributes struct {
	TaxID     string `json:"tax_id"`
	SchemeID  string `json:"scheme_id"`
	LegalName string `json:"legal_name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type legalEntityResource struct {
	Type       string                `json:"type"`
	ID         string                `json:"id"`
	Attributes legalEntityAttributes `json:"attributes"`
}

func (c *Controller) ListLegalEntities(w http.ResponseWriter, r *http.Request) error {
	items, err := c.app.Queries.ListLegalEntities.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list legal entities")
	}
	data := make([]legalEntityResource, 0, len(items))
	for _, item := range items {
		data = append(data, toResource(item))
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) CreateLegalEntity(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				TaxID     string `json:"tax_id"`
				SchemeID  string `json:"scheme_id"`
				LegalName string `json:"legal_name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	entity, err := c.app.Commands.CreateLegalEntity.Execute(r.Context(), commands.CreateLegalEntityInput{
		TaxID:     req.Data.Attributes.TaxID,
		SchemeID:  req.Data.Attributes.SchemeID,
		LegalName: req.Data.Attributes.LegalName,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toResource(*entity)})
}

func (c *Controller) UpdateLegalEntity(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				TaxID     *string `json:"tax_id"`
				SchemeID  *string `json:"scheme_id"`
				LegalName *string `json:"legal_name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	entity, err := c.app.Commands.UpdateLegalEntity.Execute(r.Context(), commands.UpdateLegalEntityInput{
		ID:        r.PathValue("id"),
		TaxID:     req.Data.Attributes.TaxID,
		SchemeID:  req.Data.Attributes.SchemeID,
		LegalName: req.Data.Attributes.LegalName,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toResource(*entity)})
}

func toResource(entity domain.LegalEntity) legalEntityResource {
	return legalEntityResource{
		Type: "legal-entities",
		ID:   entity.ID,
		Attributes: legalEntityAttributes{
			TaxID:     entity.TaxID,
			SchemeID:  entity.SchemeID,
			LegalName: entity.LegalName,
			CreatedAt: entity.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: entity.UpdatedAt.UTC().Format(time.RFC3339),
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
	mux.Handle("GET /api/v1/legal-entities", authMiddleware(api.Wrap(h.controller.ListLegalEntities, cfg)))
	mux.Handle("POST /api/v1/legal-entities", authMiddleware(api.Wrap(h.controller.CreateLegalEntity, cfg)))
	mux.Handle("PATCH /api/v1/legal-entities/{id}", authMiddleware(api.Wrap(h.controller.UpdateLegalEntity, cfg)))
}
