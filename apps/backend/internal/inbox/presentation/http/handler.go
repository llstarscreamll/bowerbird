package http

import (
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	listAccountHealthUseCase *application.ListAccountHealthUseCase
	listMessagesUseCase      *application.ListMessagesUseCase
}

func NewHandler(listAccountHealthUseCase *application.ListAccountHealthUseCase, listMessagesUseCase *application.ListMessagesUseCase) *Handler {
	return &Handler{
		listAccountHealthUseCase: listAccountHealthUseCase,
		listMessagesUseCase:      listMessagesUseCase,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("GET /api/v1/inbox/accounts/health", authMiddleware(api.Wrap(h.ListAccountHealth, isDev)))
	mux.Handle("GET /api/v1/inbox/messages", authMiddleware(api.Wrap(h.ListMessages, isDev)))
}

func (h *Handler) ListAccountHealth(w http.ResponseWriter, r *http.Request) error {
	summaries, err := h.listAccountHealthUseCase.Execute(r.Context())
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to list account health")
	}

	if len(summaries) == 0 {
		return api.Success(w, http.StatusOK, []application.AccountHealthSummary{})
	}
	return api.Success(w, http.StatusOK, summaries)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) error {
	messages, err := h.listMessagesUseCase.Execute(r.Context())
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to list messages")
	}

	if len(messages) == 0 {
		return api.Success(w, http.StatusOK, []application.UnifiedMessageSummary{})
	}
	return api.Success(w, http.StatusOK, messages)
}
