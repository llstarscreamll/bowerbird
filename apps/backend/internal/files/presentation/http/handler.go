package http

import (
	"encoding/json"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/files/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	appErrors "github.com/money-path/bowerbird/apps/backend/internal/platform/errors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	requestUploadURLCommand   *application.RequestUploadURLCommand
	requestDownloadURLCommand *application.RequestDownloadURLCommand
}

type requestUploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Module      string `json:"module"`
}

type requestDownloadURLRequest struct {
	Key string `json:"key"`
}

func NewHandler(requestUploadURLCommand *application.RequestUploadURLCommand, requestDownloadURLCommand *application.RequestDownloadURLCommand) *Handler {
	if requestUploadURLCommand == nil {
		panic("request upload url command is not configured")
	}

	if requestDownloadURLCommand == nil {
		panic("request download url command is not configured")
	}

	return &Handler{
		requestUploadURLCommand:   requestUploadURLCommand,
		requestDownloadURLCommand: requestDownloadURLCommand,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("POST /api/v1/files/uploads/presigned", authMiddleware(api.Wrap(h.RequestUploadURL, isDev)))
	mux.Handle("POST /api/v1/files/downloads/presigned", authMiddleware(api.Wrap(h.RequestDownloadURL, isDev)))
}

func (h *Handler) RequestUploadURL(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	var req requestUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	result, err := h.requestUploadURLCommand.Execute(r.Context(), application.RequestUploadURLInput{
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

func (h *Handler) RequestDownloadURL(w http.ResponseWriter, r *http.Request) error {
	var req requestDownloadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	result, err := h.requestDownloadURLCommand.Execute(r.Context(), application.RequestDownloadURLInput{
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
