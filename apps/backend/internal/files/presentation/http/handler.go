package http

import (
	"encoding/json"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/files/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	appErrors "github.com/money-path/bowerbird/apps/backend/internal/platform/errors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type Handler struct {
	requestUploadURLUseCase *application.RequestUploadURLUseCase
}

type requestUploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Module      string `json:"module"`
}

func NewHandler(requestUploadURLUseCase *application.RequestUploadURLUseCase) *Handler {
	return &Handler{requestUploadURLUseCase: requestUploadURLUseCase}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("POST /api/v1/files/uploads/presigned", authMiddleware(api.Wrap(h.RequestUploadURL, isDev)))
}

func (h *Handler) RequestUploadURL(w http.ResponseWriter, r *http.Request) error {
	if h.requestUploadURLUseCase == nil {
		return appErrors.New(appErrors.CodeInternal, "upload service is not configured")
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	tenantID, err := tenant.TenantIDFromContext(r.Context())
	if err != nil {
		return appErrors.New(appErrors.CodeValidation, "missing tenant context")
	}

	var req requestUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.Wrap(err, appErrors.CodeValidation, "invalid request body")
	}

	result, err := h.requestUploadURLUseCase.Execute(r.Context(), application.RequestUploadURLCommand{
		TenantID:    tenantID,
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
