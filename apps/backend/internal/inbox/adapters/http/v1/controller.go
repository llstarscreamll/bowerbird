package v1

import (
	"errors"
	"net/http"

	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxQueries "github.com/bowerbird/internal/inbox/application/queries"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type Controller struct {
	listAccountSyncStatusQuery *inboxQueries.ListAccountHealthQuery
	listMessagesUseCase        *inboxQueries.ListMessagesQuery
	getMessageQuery            *inboxQueries.GetMessageQuery
	syncAllAccountsCommand     *inboxCommands.SyncAllAccountsCommand
}

func NewController(
	listAccountHealthUseCase *inboxQueries.ListAccountHealthQuery,
	listMessagesUseCase *inboxQueries.ListMessagesQuery,
	getMessageUseCase *inboxQueries.GetMessageQuery,
	syncAllAccountsCommand *inboxCommands.SyncAllAccountsCommand,
) *Controller {
	return &Controller{
		listAccountSyncStatusQuery: listAccountHealthUseCase,
		listMessagesUseCase:        listMessagesUseCase,
		getMessageQuery:            getMessageUseCase,
		syncAllAccountsCommand:     syncAllAccountsCommand,
	}
}

func (c *Controller) Sync(w http.ResponseWriter, r *http.Request) error {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	if c.syncAllAccountsCommand == nil {
		return appErrors.New(appErrors.CodeInternal, "sync command not configured")
	}

	if err := c.syncAllAccountsCommand.Execute(r.Context(), claims.UserID); err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to execute sync all accounts command")
	}

	return api.Success(w, http.StatusAccepted, nil)
}

func (c *Controller) ListAccountSyncStatus(w http.ResponseWriter, r *http.Request) error {
	statuses, err := c.listAccountSyncStatusQuery.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list account sync statuses")
	}

	if len(statuses) == 0 {
		return api.Success(w, http.StatusOK, []inboxQueries.AccountSyncStatus{})
	}

	return api.Success(w, http.StatusOK, statuses)
}

func (c *Controller) ListMessages(w http.ResponseWriter, r *http.Request) error {
	messages, err := c.listMessagesUseCase.Execute(r.Context())
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list messages")
	}

	if len(messages) == 0 {
		return api.Success(w, http.StatusOK, []inboxQueries.MessageSummary{})
	}

	return api.Success(w, http.StatusOK, messages)
}

func (c *Controller) GetMessage(w http.ResponseWriter, r *http.Request) error {
	messageID := r.PathValue("messageID")
	if messageID == "" {
		return appErrors.New(appErrors.CodeValidation, "message id is required")
	}

	message, err := c.getMessageQuery.Execute(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, domain.ErrInboxMessageNotFound) {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "message not found")
		}

		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to get message")
	}

	return api.Success(w, http.StatusOK, message)
}
