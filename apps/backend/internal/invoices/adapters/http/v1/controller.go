package v1

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

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
