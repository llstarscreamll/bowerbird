package v1

import (
	"encoding/json"
	"net/http"

	"github.com/bowerbird/internal/files/application"
	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	requestUploadURLCommand   *application.RequestUploadURLCommand
	requestDownloadURLCommand *application.RequestDownloadURLCommand
}

func NewController(requestUploadURLCommand *application.RequestUploadURLCommand, requestDownloadURLCommand *application.RequestDownloadURLCommand) *Controller {
	if requestUploadURLCommand == nil {
		panic("request upload url command is not configured")
	}

	if requestDownloadURLCommand == nil {
		panic("request download url command is not configured")
	}

	return &Controller{
		requestUploadURLCommand:   requestUploadURLCommand,
		requestDownloadURLCommand: requestDownloadURLCommand,
	}
}

func (c *Controller) RequestUploadURL(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	var req requestUploadURLRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	if err := req.Validate(); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	result, err := c.requestUploadURLCommand.Execute(r.Context(), application.RequestUploadURLInput{
		UserID:      claims.UserID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		Module:      req.Module,
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "failed to create upload url")
	}

	return api.Success(w, http.StatusOK, result)
}

func (c *Controller) RequestDownloadURL(w http.ResponseWriter, r *http.Request) error {
	var req requestDownloadURLRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	if err := req.Validate(); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	result, err := c.requestDownloadURLCommand.Execute(r.Context(), application.RequestDownloadURLInput{
		Key: req.Key,
	})
	if err != nil {
		if err == application.ErrFileNotFound {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "file not found")
		}
		return appErrors.Wrap(err, appErrors.CodeValidation, "failed to create download url")
	}

	return api.Success(w, http.StatusOK, result)
}
