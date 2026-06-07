package v1

import (
	"encoding/json"
	"net/http"

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

	input := commands.QueueInvoiceExtractionFromFilesInput{Files: files}
	result, err := c.app.Commands.QueueInvoiceExtractionFromFiles.Execute(r.Context(), input)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to queue invoice extraction")
	}

	resp := newQueueInvoiceExtractionResponse(result)
	return api.Success(w, http.StatusAccepted, resp)
}
