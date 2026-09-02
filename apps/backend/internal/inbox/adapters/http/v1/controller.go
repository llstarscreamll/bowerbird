package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	entitlementsDomain "github.com/bowerbird/internal/entitlements/domain"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/application/ports"
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
	modifyMessageCommand       *inboxCommands.ModifyMessageCommand
	sendMessageCommand         *inboxCommands.SendMessageCommand
	downloadAttachmentCommand  *inboxCommands.DownloadAttachmentCommand
	features                   ports.FeatureChecker
}

func NewController(
	listAccountHealthUseCase *inboxQueries.ListAccountHealthQuery,
	listMessagesUseCase *inboxQueries.ListMessagesQuery,
	getMessageUseCase *inboxQueries.GetMessageQuery,
	syncAllAccountsCommand *inboxCommands.SyncAllAccountsCommand,
	modifyMessageCommand *inboxCommands.ModifyMessageCommand,
	sendMessageCommand *inboxCommands.SendMessageCommand,
	downloadAttachmentCommand *inboxCommands.DownloadAttachmentCommand,
	features ports.FeatureChecker,
) *Controller {
	return &Controller{
		listAccountSyncStatusQuery: listAccountHealthUseCase,
		listMessagesUseCase:        listMessagesUseCase,
		getMessageQuery:            getMessageUseCase,
		syncAllAccountsCommand:     syncAllAccountsCommand,
		modifyMessageCommand:       modifyMessageCommand,
		sendMessageCommand:         sendMessageCommand,
		downloadAttachmentCommand:  downloadAttachmentCommand,
		features:                   features,
	}
}

func (c *Controller) requireMailInbox(ctx context.Context) error {
	if c.features == nil {
		return nil
	}
	return c.features.Require(ctx, entitlementsDomain.FeatureMailInbox)
}

func (c *Controller) requireMailSend(ctx context.Context) error {
	if c.features == nil {
		return nil
	}
	return c.features.Require(ctx, entitlementsDomain.FeatureMailSend)
}

func (c *Controller) requireSyncAccess(ctx context.Context) error {
	if c.features == nil {
		return nil
	}
	return c.features.RequireAny(ctx, entitlementsDomain.FeatureMailInbox, entitlementsDomain.FeatureInvoicingCaptureFromEmail)
}

func (c *Controller) Sync(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireSyncAccess(r.Context()); err != nil {
		return err
	}

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

	return api.Success(w, http.StatusAccepted, map[string]string{"message": "Sync triggered"})
}

func (c *Controller) ListAccountSyncStatus(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireMailInbox(r.Context()); err != nil {
		return err
	}

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
	if err := c.requireMailInbox(r.Context()); err != nil {
		return err
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	onlyInvoices := query.Get("only_invoices") == "true" || query.Get("only_invoices") == "1"

	result, err := c.listMessagesUseCase.Execute(r.Context(), ports.ListMessagesFilter{
		ViewerUserID: claims.UserID,
		AccountID:    query.Get("account_id"),
		Folder:       query.Get("folder"),
		Query:        query.Get("q"),
		Limit:        limit,
		Offset:       offset,
		OnlyInvoices: onlyInvoices,
	})
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to list messages")
	}

	return api.Success(w, http.StatusOK, result)
}

func (c *Controller) GetMessage(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireMailInbox(r.Context()); err != nil {
		return err
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	messageID := r.PathValue("messageID")
	if messageID == "" {
		return appErrors.New(appErrors.CodeValidation, "message id is required")
	}

	message, err := c.getMessageQuery.Execute(r.Context(), messageID, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrInboxMessageNotFound) {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "message not found")
		}

		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to get message")
	}

	return api.Success(w, http.StatusOK, message)
}

func (c *Controller) ModifyMessage(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireMailInbox(r.Context()); err != nil {
		return err
	}

	if c.modifyMessageCommand == nil {
		return appErrors.New(appErrors.CodeInternal, "modify command not configured")
	}

	messageID := r.PathValue("messageID")
	action := inboxCommands.MessageAction(r.PathValue("action"))
	if messageID == "" {
		return appErrors.New(appErrors.CodeValidation, "message id is required")
	}

	if err := c.modifyMessageCommand.Execute(r.Context(), messageID, action); err != nil {
		if errors.Is(err, domain.ErrInboxMessageNotFound) {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "message not found")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to modify message")
	}

	return api.Success(w, http.StatusOK, map[string]string{"status": "ok"})
}

type sendMessageRequest struct {
	AccountID string   `json:"account_id"`
	To        []string `json:"to"`
	Cc        []string `json:"cc"`
	Bcc       []string `json:"bcc"`
	Subject   string   `json:"subject"`
	BodyText  string   `json:"body_text"`
	BodyHTML  string   `json:"body_html"`
	ThreadID  string   `json:"thread_id"`
	InReplyTo string   `json:"in_reply_to"`
}

func (c *Controller) SendMessage(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireMailSend(r.Context()); err != nil {
		return err
	}

	if c.sendMessageCommand == nil {
		return appErrors.New(appErrors.CodeInternal, "send command not configured")
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrors.New(appErrors.CodeValidation, "invalid request")
	}

	message, err := c.sendMessageCommand.Execute(r.Context(), inboxCommands.SendMessageInput{
		AccountID: req.AccountID,
		To:        req.To,
		Cc:        req.Cc,
		Bcc:       req.Bcc,
		Subject:   req.Subject,
		BodyText:  req.BodyText,
		BodyHTML:  req.BodyHTML,
		ThreadID:  req.ThreadID,
		InReplyTo: req.InReplyTo,
	})
	if err != nil {
		if errors.Is(err, domain.ErrOutgoingMailToRequired) {
			return appErrors.Wrap(err, appErrors.CodeValidation, "at least one recipient is required")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to send message")
	}

	return api.Success(w, http.StatusAccepted, map[string]string{"id": message.ID()})
}

func (c *Controller) DownloadAttachment(w http.ResponseWriter, r *http.Request) error {
	if err := c.requireMailInbox(r.Context()); err != nil {
		return err
	}

	if c.downloadAttachmentCommand == nil {
		return appErrors.New(appErrors.CodeInternal, "download command not configured")
	}

	messageID := r.PathValue("messageID")
	attachmentID := r.PathValue("attachmentID")
	result, err := c.downloadAttachmentCommand.Execute(r.Context(), messageID, attachmentID)
	if err != nil {
		if errors.Is(err, domain.ErrInboxMessageNotFound) {
			return appErrors.Wrap(err, appErrors.CodeNotFound, "attachment not found")
		}
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to download attachment")
	}

	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(result.Filename, `"`, "")+`"`)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(result.Data)
	return err
}
