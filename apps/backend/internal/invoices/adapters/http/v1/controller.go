package v1

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bowerbird/internal/invoices/application"
	"github.com/bowerbird/internal/invoices/application/commands"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	app *application.Application
}

func NewController(app *application.Application) *Controller {
	if app == nil {
		panic("application is required")
	}

	return &Controller{app: app}
}

func (c *Controller) QueueInvoiceExtractionFromUploadedFiles(w http.ResponseWriter, r *http.Request) error {
	var req queueInvoiceExtractionRequestDocument
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	if err := req.Validate(); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	files := make([]commands.File, 0, len(req.Data.Attributes.Files))
	for _, file := range req.Data.Attributes.Files {
		files = append(files, commands.File{
			Name:     file.Name,
			Path:     file.Path,
			MimeType: file.MimeType,
		})
	}

	input := commands.QueueInvoiceExtractionFromFilesInput{ID: req.Data.ID, Files: files}
	result, err := c.app.Commands.QueueInvoiceExtractionFromFiles.Execute(r.Context(), input)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to queue invoice extraction")
	}

	resp := newQueueInvoiceExtractionResponse(result)
	return api.Success(w, http.StatusAccepted, resp)
}

func (c *Controller) ListInvoices(w http.ResponseWriter, r *http.Request) error {
	limitStr := r.URL.Query().Get("limit")
	cursor := r.URL.Query().Get("cursor")

	limit := 20
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			return appErrors.Wrap(err, appErrors.CodeValidation, "invalid limit format, expected an integer")
		}
		limit = parsedLimit
	}

	if cursor != "" {
		match, _ := regexp.MatchString(`^[0-9A-HJKMNPQRSTVWXYZ]{26}$`, cursor)
		if !match {
			return appErrors.Wrap(nil, appErrors.CodeValidation, "invalid cursor format")
		}
	}

	result, err := c.app.Queries.ListInvoices.Execute(r.Context(), limit, cursor)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list invoices")
	}

	resp := newInvoiceListResponse(result)
	return api.Success(w, http.StatusOK, resp)
}

func (c *Controller) GetInvoiceByID(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return appErrors.Wrap(nil, appErrors.CodeValidation, "invoice id is required")
	}

	result, err := c.app.Queries.GetInvoiceByID.Execute(r.Context(), id)
	if err != nil {
		if err.Error() == "invoice not found" || (err != nil && err.Error() == "no rows in result set") {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "invoice not found")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to get invoice details")
	}

	resp := newInvoiceDetailsResponse(result)
	return api.Success(w, http.StatusOK, resp)
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
			"type": "invoice_lines",
			"id":   line.LineID,
			"attributes": map[string]any{
				"line_number": line.LineNumber,
				"item_code":   line.ItemCode,
				"description": line.Description,
				"item_id":     nullIfEmptyStr(line.ItemID),
				"link_status": line.LinkStatus,
				"link_method": nullIfEmptyStr(line.LinkMethod),
				"link_locked": line.LinkLocked,
				"suggestions": suggestions,
			},
			"relationships": map[string]any{
				"invoice": map[string]any{
					"data": map[string]any{
						"type": "invoices",
						"id":   line.InvoiceHeaderID,
					},
				},
			},
		})
	}
	return api.Success(w, http.StatusOK, map[string]any{"data": data})
}

func (c *Controller) ApplyLineDecision(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Data struct {
			Type       string `json:"type"`
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
	if strings.TrimSpace(req.Data.Type) != "" && req.Data.Type != "invoice_line_decisions" {
		return appErrors.New(appErrors.CodeValidation, "data.type must be invoice_line_decisions")
	}
	if err := c.app.Commands.ApplyLineDecision.Execute(r.Context(), commands.ApplyLineDecisionInput{
		InvoiceID: r.PathValue("invoiceId"),
		LineID:    r.PathValue("lineId"),
		ItemID:    req.Data.Attributes.ItemID,
		Action:    req.Data.Attributes.Action,
		Remember:  req.Data.Attributes.Remember,
		Lock:      req.Data.Attributes.Lock,
	}); err != nil {
		return err
	}
	return api.Success(w, http.StatusNoContent, nil)
}
