package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
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
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Stockable *bool  `json:"stockable"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type itemResource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes itemAttributes `json:"attributes"`
}

func (c *Controller) ListItems(w http.ResponseWriter, r *http.Request) error {
	items, err := c.app.Queries.ListItems.Execute(r.Context(), ports.ItemListFilter{
		Kind:   r.URL.Query().Get("kind"),
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
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

func (c *Controller) ListReviewQueue(w http.ResponseWriter, r *http.Request) error {
	lines, err := c.app.Queries.ListReviewQueue.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list review queue")
	}
	data := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		suggestions := make([]map[string]any, 0, len(line.Suggestions))
		for _, s := range line.Suggestions {
			suggestions = append(suggestions, map[string]any{
				"item_id": s.ItemID,
				"name":    s.Name,
				"score":   s.Score,
				"reason":  s.Reason,
			})
		}
		data = append(data, map[string]any{
			"type": "catalog_review_lines",
			"id":   line.LineID,
			"attributes": map[string]any{
				"invoice_header_id": line.InvoiceHeaderID,
				"line_number":       line.LineNumber,
				"item_code":         line.ItemCode,
				"description":       line.Description,
				"item_id":           nullIfEmptyStr(line.ItemID),
				"link_status":       line.LinkStatus,
				"link_method":       nullIfEmptyStr(line.LinkMethod),
				"link_locked":       line.LinkLocked,
				"suggestions":       suggestions,
			},
		})
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) RememberLineDecision(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Attributes struct {
				ItemID   string `json:"item_id"`
				Action   string `json:"action"`
				Remember bool   `json:"remember"`
				Lock     bool   `json:"lock"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}
	if err := c.app.Commands.RememberDecision.Execute(r.Context(), commands.RememberDecisionInput{
		LineID:   r.PathValue("lineId"),
		ItemID:   req.Data.Attributes.ItemID,
		Action:   req.Data.Attributes.Action,
		Remember: req.Data.Attributes.Remember,
		Lock:     req.Data.Attributes.Lock,
	}); err != nil {
		return err
	}
	return api.Success(w, http.StatusNoContent, nil)
}

func toItemResource(item domain.Item) itemResource {
	return itemResource{
		Type: "catalog_items",
		ID:   item.ID,
		Attributes: itemAttributes{
			Name:      item.Name,
			Kind:      item.Kind,
			Status:    item.Status,
			Stockable: item.Stockable,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}
}

func nullIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type Router struct {
	controller *Controller
}

func NewRouter(controller *Controller) *Router {
	return &Router{controller: controller}
}

func (h *Router) Register(mux *http.ServeMux, cfg config.Config, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/catalog/items", authMiddleware(api.Wrap(h.controller.ListItems, cfg)))
	mux.Handle("GET /api/v1/catalog/items/{id}", authMiddleware(api.Wrap(h.controller.GetItem, cfg)))
	mux.Handle("GET /api/v1/catalog/review-queue", authMiddleware(api.Wrap(h.controller.ListReviewQueue, cfg)))
	mux.Handle("POST /api/v1/catalog/lines/{lineId}/decisions", authMiddleware(api.Wrap(h.controller.RememberLineDecision, cfg)))
}
