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
	"github.com/bowerbird/internal/catalog/domain"
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
	Name           string            `json:"name"`
	Kind           string            `json:"kind"`
	Status         string            `json:"status"`
	CreationSource string            `json:"creation_source"`
	InternalCode   *string           `json:"internal_code"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	Aliases        []aliasAttributes `json:"aliases,omitempty"`
}

type aliasAttributes struct {
	ID      string  `json:"id"`
	Scheme  string  `json:"scheme"`
	Value   string  `json:"value"`
	PartyID *string `json:"party_id"`
	Source  string  `json:"source"`
}

type itemResource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes itemAttributes `json:"attributes"`
}

type aliasResource struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Attributes aliasAttributes `json:"attributes"`
}

func (c *Controller) ListItems(w http.ResponseWriter, r *http.Request) error {
	limit := pageSize(r, 50, 100)
	afterName, afterID := decodeItemCursor(pageAfter(r))
	page, err := c.app.Queries.ListItems.Execute(r.Context(), ports.ItemListFilter{
		Kind:           r.URL.Query().Get("kind"),
		Status:         r.URL.Query().Get("status"),
		Search:         r.URL.Query().Get("search"),
		CreationSource: r.URL.Query().Get("creation_source"),
		Limit:          limit,
		AfterName:      afterName,
		AfterID:        afterID,
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list catalog items")
	}
	data := make([]itemResource, 0, len(page.Items))
	for _, item := range page.Items {
		data = append(data, toItemResource(item))
	}
	cursor := ""
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		cursor = encodeCursor(last.Name, last.ID)
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data, "meta": pageMeta(page.HasMore, cursor, nil)})
}

func (c *Controller) GetItem(w http.ResponseWriter, r *http.Request) error {
	detail, err := c.app.Queries.GetItemByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toItemDetailResource(*detail)})
}

func (c *Controller) CreateItem(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Name         string `json:"name"`
				Kind         string `json:"kind"`
				InternalCode string `json:"internal_code"`
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
		ID:           req.Data.ID,
		Name:         req.Data.Attributes.Name,
		Kind:         req.Data.Attributes.Kind,
		InternalCode: req.Data.Attributes.InternalCode,
	}); err != nil {
		return err
	}
	detail, err := c.app.Queries.GetItemByID.Execute(r.Context(), req.Data.ID)
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toItemDetailResource(*detail)})
}

func (c *Controller) UpdateItem(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				Name         *string `json:"name"`
				Kind         *string `json:"kind"`
				Status       *string `json:"status"`
				InternalCode *string `json:"internal_code"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if err := c.app.Commands.UpdateItem.Execute(r.Context(), commands.UpdateItemInput{
		ID:           r.PathValue("id"),
		Name:         req.Data.Attributes.Name,
		Kind:         req.Data.Attributes.Kind,
		Status:       req.Data.Attributes.Status,
		InternalCode: req.Data.Attributes.InternalCode,
	}); err != nil {
		return err
	}
	detail, err := c.app.Queries.GetItemByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": toItemDetailResource(*detail)})
}

func (c *Controller) AddItemAlias(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Scheme  string `json:"scheme"`
				Value   string `json:"value"`
				PartyID string `json:"party_id"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "catalog_item_aliases" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be catalog_item_aliases")
	}
	if err := c.app.Commands.AddItemAlias.Execute(r.Context(), commands.AddItemAliasInput{
		ItemID:  r.PathValue("id"),
		AliasID: req.Data.ID,
		Scheme:  req.Data.Attributes.Scheme,
		Value:   req.Data.Attributes.Value,
		PartyID: req.Data.Attributes.PartyID,
	}); err != nil {
		return err
	}
	detail, err := c.app.Queries.GetItemByID.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	var created *domain.Alias
	for i := range detail.Aliases {
		if detail.Aliases[i].ID == req.Data.ID {
			created = &detail.Aliases[i]
			break
		}
	}
	if created == nil {
		return appErrors.New(appErrors.CodeInternal, "alias was not persisted")
	}
	return api.Success(w, http.StatusCreated, map[string]any{"data": toAliasResource(*created)})
}

func (c *Controller) RemoveItemAlias(w http.ResponseWriter, r *http.Request) error {
	if err := c.app.Commands.RemoveItemAlias.Execute(r.Context(), r.PathValue("id"), r.PathValue("aliasId")); err != nil {
		return err
	}
	return api.Success(w, http.StatusNoContent, nil)
}

func toItemResource(item domain.Item) itemResource {
	return toItemDetailResource(queries.ItemDetail{Item: item, Aliases: nil})
}

func toItemDetailResource(detail queries.ItemDetail) itemResource {
	item := detail.Item
	var code *string
	if parsed, ok := item.ParsedInternalCode(); ok {
		s := parsed.String()
		code = &s
	}
	aliases := make([]aliasAttributes, 0, len(detail.Aliases))
	for _, alias := range detail.Aliases {
		aliases = append(aliases, toAliasAttributes(alias))
	}
	attrs := itemAttributes{
		Name:           item.Name,
		Kind:           item.Kind,
		Status:         item.Status,
		CreationSource: item.CreationSource,
		InternalCode:   code,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if len(aliases) > 0 {
		attrs.Aliases = aliases
	}
	return itemResource{
		Type:       "catalog_items",
		ID:         item.ID,
		Attributes: attrs,
	}
}

func toAliasAttributes(alias domain.Alias) aliasAttributes {
	return aliasAttributes{
		ID:      alias.ID,
		Scheme:  alias.Scheme,
		Value:   alias.Value,
		PartyID: alias.PartyID,
		Source:  alias.Source,
	}
}

func toAliasResource(alias domain.Alias) aliasResource {
	return aliasResource{
		Type:       "catalog_item_aliases",
		ID:         alias.ID,
		Attributes: toAliasAttributes(alias),
	}
}

type Router struct {
	controller *Controller
}

func NewRouter(controller *Controller) *Router {
	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/catalog/imports/template", authMiddleware(api.Wrap(h.controller.DownloadImportTemplate, cfg)))
	mux.Handle("GET /api/v1/catalog/imports", authMiddleware(api.Wrap(h.controller.ListImports, cfg)))
	mux.Handle("POST /api/v1/catalog/imports", authMiddleware(api.Wrap(h.controller.CreateImport, cfg)))
	mux.Handle("GET /api/v1/catalog/imports/{id}", authMiddleware(api.Wrap(h.controller.GetImport, cfg)))
	mux.Handle("POST /api/v1/catalog/imports/{id}/cancel", authMiddleware(api.Wrap(h.controller.CancelImport, cfg)))
	mux.Handle("GET /api/v1/catalog/imports/{id}/errors", authMiddleware(api.Wrap(h.controller.ListImportErrors, cfg)))
	mux.Handle("GET /api/v1/catalog/items", authMiddleware(api.Wrap(h.controller.ListItems, cfg)))
	mux.Handle("POST /api/v1/catalog/items", authMiddleware(api.Wrap(h.controller.CreateItem, cfg)))
	mux.Handle("GET /api/v1/catalog/items/{id}", authMiddleware(api.Wrap(h.controller.GetItem, cfg)))
	mux.Handle("PATCH /api/v1/catalog/items/{id}", authMiddleware(api.Wrap(h.controller.UpdateItem, cfg)))
	mux.Handle("POST /api/v1/catalog/items/{id}/aliases", authMiddleware(api.Wrap(h.controller.AddItemAlias, cfg)))
	mux.Handle("DELETE /api/v1/catalog/items/{id}/aliases/{aliasId}", authMiddleware(api.Wrap(h.controller.RemoveItemAlias, cfg)))
}
