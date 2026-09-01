package v1

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/application/queries"
	"github.com/bowerbird/internal/platform/config"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	app *application.Application
}

func NewController(app *application.Application) *Controller {
	if app == nil {
		panic("catalog application is required")
	}
	return &Controller{app: app}
}

type itemAttributes struct {
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Status         string  `json:"status"`
	CreationSource string  `json:"creation_source"`
	InternalSKU    *string `json:"internal_sku"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type itemResource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes itemAttributes `json:"attributes"`
}

func (c *Controller) ListItems(w http.ResponseWriter, r *http.Request) error {
	items, err := c.app.Queries.ListItems.Execute(r.Context(), ports.ItemListFilter{
		Kind:           r.URL.Query().Get("kind"),
		Status:         r.URL.Query().Get("status"),
		Search:         r.URL.Query().Get("search"),
		CreationSource: r.URL.Query().Get("creation_source"),
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list catalog items")
	}
	data := make([]itemResource, 0, len(items))
	for _, item := range items {
		data = append(data, toItemResource(item))
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) GetItem(w http.ResponseWriter, r *http.Request) error {
	item, err := c.app.Queries.GetItemByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toItemResource(*item)})
}

func (c *Controller) CreateItem(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Name        string `json:"name"`
				Kind        string `json:"kind"`
				InternalSKU string `json:"internal_sku"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "catalog_items" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be catalog_items")
	}
	if err := c.app.Commands.CreateItem.Execute(r.Context(), commands.CreateItemInput{
		ID:          req.Data.ID,
		Name:        req.Data.Attributes.Name,
		Kind:        req.Data.Attributes.Kind,
		InternalSKU: req.Data.Attributes.InternalSKU,
	}); err != nil {
		return err
	}
	view, err := c.app.Queries.GetItemByID.Execute(r.Context(), req.Data.ID)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toItemResource(*view)})
}

func (c *Controller) UpdateItem(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Name        *string `json:"name"`
				Kind        *string `json:"kind"`
				Status      *string `json:"status"`
				InternalSKU *string `json:"internal_sku"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if err := c.app.Commands.UpdateItem.Execute(r.Context(), commands.UpdateItemInput{
		ID:          r.PathValue("id"),
		Name:        req.Data.Attributes.Name,
		Kind:        req.Data.Attributes.Kind,
		Status:      req.Data.Attributes.Status,
		InternalSKU: req.Data.Attributes.InternalSKU,
	}); err != nil {
		return err
	}
	view, err := c.app.Queries.GetItemByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toItemResource(*view)})
}

func toItemResource(view queries.ItemView) itemResource {
	return itemResource{
		Type: "catalog_items",
		ID:   view.Item.ID,
		Attributes: itemAttributes{
			Name:           view.Item.Name,
			Kind:           view.Item.Kind,
			Status:         view.Item.Status,
			CreationSource: view.Item.CreationSource,
			InternalSKU:    view.InternalSKU,
			CreatedAt:      view.Item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      view.Item.UpdatedAt.UTC().Format(time.RFC3339),
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
	mux.Handle("GET /api/v1/catalog/items", authMiddleware(api.Wrap(h.controller.ListItems, cfg)))
	mux.Handle("POST /api/v1/catalog/items", authMiddleware(api.Wrap(h.controller.CreateItem, cfg)))
	mux.Handle("GET /api/v1/catalog/items/{id}", authMiddleware(api.Wrap(h.controller.GetItem, cfg)))
	mux.Handle("PATCH /api/v1/catalog/items/{id}", authMiddleware(api.Wrap(h.controller.UpdateItem, cfg)))
}
