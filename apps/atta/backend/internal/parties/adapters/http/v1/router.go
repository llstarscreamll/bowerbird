package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/atta/internal/parties/application"
	"github.com/atta/internal/parties/application/commands"
	"github.com/atta/internal/parties/application/ports"
	"github.com/atta/internal/parties/domain"
	"github.com/atta/internal/platform/config"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/http/api"
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

type emailAttributes struct {
	ID     string `json:"id"`
	Value  string `json:"value"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

type phoneAttributes struct {
	ID     string `json:"id"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

type addressAttributes struct {
	ID          string `json:"id"`
	Line        string `json:"line"`
	City        string `json:"city"`
	Department  string `json:"department"`
	PostalZone  string `json:"postal_zone"`
	CountryCode string `json:"country_code"`
	Kind        string `json:"kind"`
	Source      string `json:"source"`
}

type partyAttributes struct {
	TaxID          string              `json:"tax_id"`
	SchemeID       string              `json:"scheme_id"`
	TaxpayerKind   string              `json:"taxpayer_kind"`
	TaxLevelCodes  []string            `json:"tax_level_codes"`
	Name           string              `json:"name"`
	Roles          []string            `json:"roles"`
	Status         string              `json:"status"`
	CreationSource string              `json:"creation_source"`
	Emails         []emailAttributes   `json:"emails"`
	Phones         []phoneAttributes   `json:"phones"`
	Addresses      []addressAttributes `json:"addresses"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
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
				Name     string   `json:"name"`
				TaxID    string   `json:"tax_id"`
				SchemeID string   `json:"scheme_id"`
				Roles    []string `json:"roles"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.CreateParty.Execute(r.Context(), commands.CreatePartyInput{
		Name:     req.Data.Attributes.Name,
		TaxID:    req.Data.Attributes.TaxID,
		SchemeID: req.Data.Attributes.SchemeID,
		Roles:    req.Data.Attributes.Roles,
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
				Name         *string   `json:"name"`
				Roles        *[]string `json:"roles"`
				SchemeID     *string   `json:"scheme_id"`
				TaxpayerKind *string   `json:"taxpayer_kind"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.UpdateParty.Execute(r.Context(), commands.UpdatePartyInput{
		ID:           r.PathValue("id"),
		Name:         req.Data.Attributes.Name,
		Roles:        req.Data.Attributes.Roles,
		SchemeID:     req.Data.Attributes.SchemeID,
		TaxpayerKind: req.Data.Attributes.TaxpayerKind,
	})
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) AddEmail(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.PartyChannels.AddEmail(r.Context(), r.PathValue("id"), req.Data.Attributes.Value)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) AddPhone(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	party, err := c.app.Commands.PartyChannels.AddPhone(r.Context(), r.PathValue("id"), req.Data.Attributes.Value)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) AddAddress(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Line        string `json:"line"`
				City        string `json:"city"`
				Department  string `json:"department"`
				PostalZone  string `json:"postal_zone"`
				CountryCode string `json:"country_code"`
				Kind        string `json:"kind"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	a := req.Data.Attributes
	party, err := c.app.Commands.PartyChannels.AddAddress(r.Context(), r.PathValue("id"), a.Line, a.City, a.Department, a.PostalZone, a.CountryCode, a.Kind)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) DeleteEmail(w http.ResponseWriter, r *http.Request) error {
	party, err := c.app.Commands.PartyChannels.RemoveEmail(r.Context(), r.PathValue("id"), r.PathValue("emailId"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) DeletePhone(w http.ResponseWriter, r *http.Request) error {
	party, err := c.app.Commands.PartyChannels.RemovePhone(r.Context(), r.PathValue("id"), r.PathValue("phoneId"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toPartyResource(*party)})
}

func (c *Controller) DeleteAddress(w http.ResponseWriter, r *http.Request) error {
	party, err := c.app.Commands.PartyChannels.RemoveAddress(r.Context(), r.PathValue("id"), r.PathValue("addressId"))
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
	emails := make([]emailAttributes, 0, len(party.Emails))
	for _, e := range party.Emails {
		emails = append(emails, emailAttributes{ID: e.ID, Value: e.Value, Kind: e.Kind, Source: e.Source})
	}
	phones := make([]phoneAttributes, 0, len(party.Phones))
	for _, p := range party.Phones {
		phones = append(phones, phoneAttributes{ID: p.ID, Value: p.Value, Source: p.Source})
	}
	addresses := make([]addressAttributes, 0, len(party.Addresses))
	for _, a := range party.Addresses {
		addresses = append(addresses, addressAttributes{
			ID: a.ID, Line: a.Line, City: a.City, Department: a.Department,
			PostalZone: a.PostalZone, CountryCode: a.CountryCode, Kind: a.Kind, Source: a.Source,
		})
	}
	codes := party.TaxLevelCodes
	if codes == nil {
		codes = []string{}
	}
	return partyResource{
		Type: "parties",
		ID:   party.ID,
		Attributes: partyAttributes{
			TaxID:          party.TaxID,
			SchemeID:       party.SchemeID,
			TaxpayerKind:   party.TaxpayerKind,
			TaxLevelCodes:  codes,
			Name:           party.Name,
			Roles:          roles,
			Status:         party.Status,
			CreationSource: party.CreationSource,
			Emails:         emails,
			Phones:         phones,
			Addresses:      addresses,
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
	mux.Handle("POST /api/v1/parties/{id}/emails", authMiddleware(api.Wrap(h.controller.AddEmail, cfg)))
	mux.Handle("DELETE /api/v1/parties/{id}/emails/{emailId}", authMiddleware(api.Wrap(h.controller.DeleteEmail, cfg)))
	mux.Handle("POST /api/v1/parties/{id}/phones", authMiddleware(api.Wrap(h.controller.AddPhone, cfg)))
	mux.Handle("DELETE /api/v1/parties/{id}/phones/{phoneId}", authMiddleware(api.Wrap(h.controller.DeletePhone, cfg)))
	mux.Handle("POST /api/v1/parties/{id}/addresses", authMiddleware(api.Wrap(h.controller.AddAddress, cfg)))
	mux.Handle("DELETE /api/v1/parties/{id}/addresses/{addressId}", authMiddleware(api.Wrap(h.controller.DeleteAddress, cfg)))
}
