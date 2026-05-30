package http

import (
	"errors"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	appErrors "github.com/money-path/bowerbird/apps/backend/internal/platform/errors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	listAccountSyncStatusQuery *application.ListAccountHealthUseCase
	listMessagesUseCase        *application.ListMessagesUseCase
	getMessageQuery            *application.GetMessageUseCase
	syncAllAccountsCommand     *application.SyncAllAccountsCommand
}

func NewHandler(
	listAccountHealthUseCase *application.ListAccountHealthUseCase,
	listMessagesUseCase *application.ListMessagesUseCase,
	getMessageUseCase *application.GetMessageUseCase,
	syncAllAccountsCommand *application.SyncAllAccountsCommand,
) *Handler {
	return &Handler{
		listAccountSyncStatusQuery: listAccountHealthUseCase,
		listMessagesUseCase:        listMessagesUseCase,
		getMessageQuery:            getMessageUseCase,
		syncAllAccountsCommand:     syncAllAccountsCommand,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("GET /api/v1/inbox/sync-status", authMiddleware(api.Wrap(h.ListAccountSyncStatus, isDev)))
	mux.Handle("GET /api/v1/inbox/messages", authMiddleware(api.Wrap(h.ListMessages, isDev)))
	mux.Handle("GET /api/v1/inbox/messages/{messageID}", authMiddleware(api.Wrap(h.GetMessage, isDev)))
	mux.Handle("POST /api/v1/inbox/sync", authMiddleware(api.Wrap(h.Sync, isDev)))
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	if h.syncAllAccountsCommand == nil {
		return appErrors.New(appErrors.CodeInternal, "sync command not configured")
	}

	if err := h.syncAllAccountsCommand.Execute(r.Context(), claims.UserID); err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to execute sync all accounts command")
	}

	return api.Success(w, http.StatusAccepted, nil)
}

func (h *Handler) ListAccountSyncStatus(w http.ResponseWriter, r *http.Request) error {
	statuses, err := h.listAccountSyncStatusQuery.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list account sync statuses")
	}

	if len(statuses) == 0 {
		return api.Success(w, http.StatusOK, []application.AccountSyncStatus{})
	}

	return api.Success(w, http.StatusOK, statuses)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) error {
	messages, err := h.listMessagesUseCase.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list messages")
	}

	if len(messages) == 0 {
		return api.Success(w, http.StatusOK, []application.UnifiedMessageSummary{})
	}
	return api.Success(w, http.StatusOK, messages)
}

func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) error {
	messageID := r.PathValue("messageID")
	if messageID == "" {
		return appErrors.New(appErrors.CodeValidation, "message id is required")
	}

	message, err := h.getMessageQuery.Execute(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, domain.ErrEmailMessageNotFound) {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "message not found")
		}

		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to get message")
	}

	return api.Success(w, http.StatusOK, message)
}
