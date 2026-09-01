package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bowerbird/internal/parties/application"
	"github.com/bowerbird/internal/parties/application/commands"
	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	app *application.Application
}

func NewController(app *application.Application) *Controller {
	if app == nil {
		panic("parties application is required")
	}
	return &Controller{app: app}
}

type partyAttributes struct {
	TaxID          string   `json:"tax_id"`
	Name           string   `json:"name"`
	Roles          []string `json:"roles"`
	Status         string   `json:"status"`
	CreationSource string   `json:"creation_source"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type partyResource struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Attributes partyAttributes `json:"attributes"`
}

func (c *Controller) ListParties(w http.ResponseWriter, r *http.Request) error {
	parties, err := c.app.Queries.ListParties.Execute(r.Context(), ports.ListFilter{
		Role:           r.URL.Query().Get("role"),
		Search:         r.URL.Query().Get("search"),
		CreationSource: r.URL.Query().Get("creation_source"),
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list parties")
	}
	data := make([]partyResource, 0, len(parties))
	for _, party := range parties {
		data = append(data, toPartyResource(party))
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) GetParty(w http.ResponseWriter, r *http.Request) error {
	party, err := c.app.Queries.GetPartyByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) CreateParty(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Name  string   `json:"name"`
				TaxID string   `json:"tax_id"`
				Roles []string `json:"roles"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.CreateParty.Execute(r.Context(), commands.CreatePartyInput{
		Name:  req.Data.Attributes.Name,
		TaxID: req.Data.Attributes.TaxID,
		Roles: req.Data.Attributes.Roles,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) UpdateParty(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Name  *string   `json:"name"`
				Roles *[]string `json:"roles"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.UpdateParty.Execute(r.Context(), commands.UpdatePartyInput{
		ID:    r.PathValue("id"),
		Name:  req.Data.Attributes.Name,
		Roles: req.Data.Attributes.Roles,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toPartyResource(*party)})
}

func toPartyResource(party domain.Party) partyResource {
	roles := party.Roles
	if roles == nil {
		roles = []string{}
	}
	return partyResource{
		Type: "parties",
		ID:   party.ID,
		Attributes: partyAttributes{
			TaxID:          party.TaxID,
			Name:           party.Name,
			Roles:          roles,
			Status:         party.Status,
			CreationSource: party.CreationSource,
			CreatedAt:      party.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      party.UpdatedAt.UTC().Format(time.RFC3339),
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
	mux.Handle("GET /api/v1/parties", authMiddleware(api.Wrap(h.controller.ListParties, cfg)))
	mux.Handle("POST /api/v1/parties", authMiddleware(api.Wrap(h.controller.CreateParty, cfg)))
	mux.Handle("GET /api/v1/parties/{id}", authMiddleware(api.Wrap(h.controller.GetParty, cfg)))
	mux.Handle("PATCH /api/v1/parties/{id}", authMiddleware(api.Wrap(h.controller.UpdateParty, cfg)))
}
