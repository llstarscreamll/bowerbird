package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/apperrors"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/http/api"
)

type Handler struct {
	listAccountHealthUseCase *application.ListAccountHealthUseCase
	listMessagesUseCase      *application.ListMessagesUseCase
	getMessageUseCase        *application.GetMessageUseCase
	triggerSyncUseCase       *application.TriggerSyncUseCase
}

func NewHandler(
	listAccountHealthUseCase *application.ListAccountHealthUseCase,
	listMessagesUseCase *application.ListMessagesUseCase,
	getMessageUseCase *application.GetMessageUseCase,
	triggerSyncUseCase *application.TriggerSyncUseCase,
) *Handler {
	return &Handler{
		listAccountHealthUseCase: listAccountHealthUseCase,
		listMessagesUseCase:      listMessagesUseCase,
		getMessageUseCase:        getMessageUseCase,
		triggerSyncUseCase:       triggerSyncUseCase,
	}
}

func (h *Handler) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, isDev bool) {
	mux.Handle("GET /api/v1/inbox/sync-status", authMiddleware(api.Wrap(h.ListAccountHealth, isDev)))
	mux.Handle("GET /api/v1/inbox/messages", authMiddleware(api.Wrap(h.ListMessages, isDev)))
	mux.Handle("GET /api/v1/inbox/messages/{messageID}", authMiddleware(api.Wrap(h.GetMessage, isDev)))
	mux.Handle("POST /api/v1/inbox/sync", authMiddleware(api.Wrap(h.TriggerSync, isDev)))
}

type triggerSyncRequest struct {
	AccountID string `json:"account_id,omitempty"`
}

func (h *Handler) TriggerSync(w http.ResponseWriter, r *http.Request) error {
	var req triggerSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Ignore EOF since body is optional
	}

	var accID *string
	if req.AccountID != "" {
		accID = &req.AccountID
	}

	if err := h.triggerSyncUseCase.Execute(r.Context(), accID); err != nil {
		if errors.Is(err, application.ErrActiveConnectionNotFound) {
			return apperrors.Wrap(err, apperrors.CodeValidation, "active connection not found")
		}
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to trigger sync")
	}

	return api.Success(w, http.StatusAccepted, map[string]string{"message": "Sync triggered"})
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

func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) error {
	messageID := r.PathValue("messageID")
	if messageID == "" {
		return apperrors.New(apperrors.CodeValidation, "message id is required")
	}

	message, err := h.getMessageUseCase.Execute(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, domain.ErrEmailMessageNotFound) {
			return apperrors.Wrap(err, apperrors.CodeNotFound, "message not found")
		}
		return apperrors.Wrap(err, apperrors.CodeInternal, "failed to get message")
	}

	return api.Success(w, http.StatusOK, message)
}
