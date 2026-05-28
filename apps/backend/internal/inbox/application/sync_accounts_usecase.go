package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/observability"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type ProviderClientFactory interface {
	Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error)
}

type InboxEventPublisher interface {
	PublishInboxMessageReceived(ctx context.Context, event domain.InboxMessageReceived) error
}

type AttachmentStorage interface {
	StoreAttachment(ctx context.Context, input StoreAttachmentInput) (*StoredAttachment, error)
}

type StoreAttachmentInput struct {
	TenantSlug         string
	ConnectedAccountID string
	MessageID          string
	AttachmentID       string
	Filename           string
	ContentType        string
	Data               []byte
}

type StoredAttachment struct {
	S3Key     string
	SHA256    string
	SizeBytes int64
	Uploaded  bool
}

type SyncAccountsUseCase struct {
	repo               domain.Repository
	connectionsService application.InternalService
	providerFactory    ProviderClientFactory
	publisher          InboxEventPublisher
	attachmentStorage  AttachmentStorage
	logger             *slog.Logger
	metrics            observability.Metrics
	now                func() time.Time
	newID              func() string
	perMessageTimeout  time.Duration
	maxRawMessageBytes int
	maxAttachmentBytes int64
}

type SyncAccountsResult struct {
	AccountsProcessed int
	MessagesSynced    int
	EventsPublished   int
	Failures          int
}

func NewSyncAccountsUseCase(
	repo domain.Repository,
	connectionsService application.InternalService,
	providerFactory ProviderClientFactory,
	publisher InboxEventPublisher,
	attachmentStorage AttachmentStorage,
) *SyncAccountsUseCase {
	return &SyncAccountsUseCase{
		repo:               repo,
		connectionsService: connectionsService,
		providerFactory:    providerFactory,
		publisher:          publisher,
		attachmentStorage:  attachmentStorage,
		logger:             slog.Default(),
		metrics:            observability.NoopMetrics{},
		now:                time.Now,
		newID:              id.NewULID,
		perMessageTimeout:  20 * time.Second,
		maxRawMessageBytes: 1 * 1024 * 1024,
		maxAttachmentBytes: 10 * 1024 * 1024,
	}
}

func (u *SyncAccountsUseCase) Run(ctx context.Context) (*SyncAccountsResult, error) {
	startedAt := u.now()
	tenantSlug, err := tenant.TenantSlugFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := u.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active connections: %w", err)
	}

	result := &SyncAccountsResult{}
	for _, account := range accounts {
		result.AccountsProcessed++
		err := u.SyncSingleAccountWithResult(ctx, tenantSlug, account, result)
		if err != nil {
			// already logged
		}
	}

	u.logger.Info(
		"sync accounts completed",
		"tenant_slug",
		tenantSlug,
		"accounts_processed",
		result.AccountsProcessed,
		"messages_synced",
		result.MessagesSynced,
		"failures",
		result.Failures,
		"duration",
		u.now().Sub(startedAt).String(),
	)

	return result, nil
}

func (u *SyncAccountsUseCase) SyncSingleAccount(ctx context.Context, connectionID string) (*SyncAccountsResult, error) {
	tenantSlug, err := tenant.TenantSlugFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := u.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active connections: %w", err)
	}

	var targetAccount *application.ConnectionInfo
	for _, acc := range accounts {
		if acc.ID == connectionID {
			targetAccount = &acc
			break
		}
	}

	if targetAccount == nil {
		return nil, fmt.Errorf("active connection not found: %s", connectionID)
	}

	result := &SyncAccountsResult{}
	result.AccountsProcessed = 1
	err = u.SyncSingleAccountWithResult(ctx, tenantSlug, *targetAccount, result)
	return result, err
}

func (u *SyncAccountsUseCase) SyncSingleAccountWithResult(ctx context.Context, tenantSlug string, account application.ConnectionInfo, result *SyncAccountsResult) error {
	cursor, err := u.repo.GetSyncCursor(ctx, account.ID)
	if err != nil {
		u.logger.Error("failed to get sync cursor", "account_id", account.ID, "error", err)
		return err
	}
	if cursor == nil {
		cursor = &domain.InboxSyncCursor{
			ConnectionID: account.ID,
			Status:       domain.InboxSyncStatusIdle,
		}
	}

	if err := u.syncAccount(ctx, tenantSlug, account, cursor, result); err != nil {
		err = classifySyncError(account, err)
		errorCode := syncErrorCode(err)
		result.Failures++
		u.metrics.IncCounter("inbox_sync_errors_total", map[string]string{"tenant_slug": tenantSlug, "provider": account.Provider, "error_code": errorCode})
		u.metrics.IncCounter("sync_error_classified_total", map[string]string{"tenant_slug": tenantSlug, "provider": account.Provider, "error_code": errorCode})
		u.logger.Info(
			"sync_error_classified",
			"tenant_slug",
			tenantSlug,
			"account_id",
			account.ID,
			"provider",
			account.Provider,
			"error_code",
			errorCode,
		)
		u.logger.Error(
			"inbox account sync failed",
			"tenant_slug",
			tenantSlug,
			"account_id",
			account.ID,
			"provider",
			account.Provider,
			"error",
			err,
		)
		now := u.now().UTC()
		cursor.MarkSyncFailed(now, err.Error())
		_ = u.repo.UpsertSyncCursor(ctx, cursor)

		if shouldMarkRequiresReconnect(err) {
			_ = u.connectionsService.MarkRequiresReconnect(ctx, account.ID, err.Error())
		}
		return err
	}

	return nil
}

func (u *SyncAccountsUseCase) syncAccount(ctx context.Context, tenantSlug string, account application.ConnectionInfo, cursor *domain.InboxSyncCursor, result *SyncAccountsResult) error {
	credentialsJSON, err := u.connectionsService.DecryptCredentials(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("decrypt account credentials: %w", err)
	}

	client, err := u.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	query := incrementalQuery(cursor.LastSyncedAt)
	pageToken := ""
	for {
		refs, nextPageToken, err := client.ListMessages(ctx, domain.ListMessagesOptions{
			UserID:     "me",
			Query:      query,
			PageToken:  pageToken,
			MaxResults: 100,
		})
		if err != nil {
			return fmt.Errorf("list provider messages: %w", err)
		}

		for _, ref := range refs {
			if err := u.processSingleMessage(ctx, tenantSlug, account, ref, client, result); err != nil {
				if errors.Is(err, errPayloadRejected) {
					u.metrics.IncCounter("sync_payload_rejected_total", map[string]string{"tenant_slug": tenantSlug, "provider": account.Provider})
					u.logger.Info(
						"sync_payload_rejected",
						"tenant_slug",
						tenantSlug,
						"account_id",
						account.ID,
						"provider",
						account.Provider,
						"provider_message_id",
						ref.ID,
					)
					u.logger.Warn(
						"message payload rejected during sync",
						"tenant_slug",
						tenantSlug,
						"account_id",
						account.ID,
						"provider",
						account.Provider,
						"provider_message_id",
						ref.ID,
						"error",
						err,
					)
					continue
				}

				return err
			}
		}

		pageToken = nextPageToken
		if pageToken == "" {
			break
		}
	}

	now := u.now().UTC()
	cursor.MarkSyncSucceeded(now)
	return u.repo.UpsertSyncCursor(ctx, cursor)
}

func (u *SyncAccountsUseCase) processSingleMessage(
	ctx context.Context,
	tenantSlug string,
	account application.ConnectionInfo,
	ref domain.MessageRef,
	client domain.MailProviderClient,
	result *SyncAccountsResult,
) (retErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			retErr = fmt.Errorf("panic while processing provider message %s: %v: %w", ref.ID, recovered, errPayloadRejected)
			u.logger.Error(
				"panic recovered during inbox sync message processing",
				"tenant_slug",
				tenantSlug,
				"account_id",
				account.ID,
				"provider",
				account.Provider,
				"provider_message_id",
				ref.ID,
				"error",
				retErr,
			)
		}
	}()

	messageCtx, cancel := context.WithTimeout(ctx, u.perMessageTimeout)
	defer cancel()

	message, err := client.GetMessage(messageCtx, "me", ref.ID)
	if err != nil {
		return fmt.Errorf("get provider message %s: %w", ref.ID, err)
	}

	if err := u.validateMessagePayload(message); err != nil {
		return fmt.Errorf("validate provider message %s: %w", ref.ID, err)
	}

	rawData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal provider message: %w", err)
	}
	if len(rawData) > u.maxRawMessageBytes {
		return fmt.Errorf("raw provider message size %d exceeds max %d: %w", len(rawData), u.maxRawMessageBytes, errPayloadRejected)
	}

	now := u.now().UTC()

	var threadIDPtr *string
	if message.ThreadID != "" {
		threadIDStr := message.ThreadID
		threadIDPtr = &threadIDStr
	}

	var subjectPtr *string
	if message.Subject != "" {
		subjectStr := message.Subject
		subjectPtr = &subjectStr
	}

	var senderPtr *string
	if message.Sender != "" {
		senderStr := message.Sender
		senderPtr = &senderStr
	}

	emailMessage, err := domain.NewSyncedEmailMessage(domain.NewEmailMessageInput{
		ID:                u.newID(),
		AccountID:         account.ID,
		ProviderMessageID: message.ID,
		ProviderThreadID:  threadIDPtr,
		Subject:           subjectPtr,
		SenderEmail:       senderPtr,
		ReceivedAt:        message.ReceivedAt,
		RawData:           rawData,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return fmt.Errorf("build internal message: %w", err)
	}

	if _, err := u.repo.UpsertEmailMessage(ctx, emailMessage); err != nil {
		return fmt.Errorf("save internal message: %w", err)
	}

	var attachmentRefs []domain.AttachmentRef
	if len(message.Attachments) > 0 {
		attachmentRefs, err = u.syncMessageAttachments(messageCtx, tenantSlug, account.ID, emailMessage.ID, message.Attachments, client)
		if err != nil {
			if errors.Is(err, errPayloadRejected) {
				return err
			}
			u.logger.Error("failed to sync attachments", "message_id", emailMessage.ID, "error", err)
		}
	}

	result.MessagesSynced++

	if u.publisher != nil {
		event := domain.InboxMessageReceived{
			EventID:           u.newID(),
			OccurredAt:        now.Format(time.RFC3339Nano),
			TenantSlug:        tenantSlug,
			AccountID:         account.ID,
			Provider:          account.Provider,
			ProviderMessageID: message.ID,
			MessageInternalID: emailMessage.ID,
		}

		if message.Subject != "" {
			event.Subject = message.Subject
		}
		if message.Sender != "" {
			event.Sender = message.Sender
		}
		if message.ReceivedAt != nil {
			event.ReceivedAt = message.ReceivedAt.Format(time.RFC3339)
		}

		for _, att := range attachmentRefs {
			event.AttachmentRefs = append(event.AttachmentRefs, domain.AttachmentRef{
				S3Key:    att.S3Key,
				Filename: att.Filename,
				MimeType: att.MimeType,
				SHA256:   att.SHA256,
			})
		}

		if err := u.publisher.PublishInboxMessageReceived(ctx, event); err != nil {
			u.logger.Error("failed to publish InboxMessageReceived", "event_id", event.EventID, "error", err)
		} else {
			result.EventsPublished++
		}
	}

	return nil
}

func (u *SyncAccountsUseCase) validateMessagePayload(message *domain.MailMessage) error {
	if err := sanitizeAndValidateMailMessage(message); err != nil {
		return fmt.Errorf("sanitize and validate message: %v: %w", err, errPayloadRejected)
	}

	if u.maxRawMessageBytes > 0 && len(message.PlainTextBody) > u.maxRawMessageBytes {
		return fmt.Errorf("plain text body exceeds max size: %w", errPayloadRejected)
	}

	for _, att := range message.Attachments {
		if u.maxAttachmentBytes > 0 && att.Size > u.maxAttachmentBytes {
			return fmt.Errorf("attachment size %d exceeds max %d: %w", att.Size, u.maxAttachmentBytes, errPayloadRejected)
		}
	}

	return nil
}

func (u *SyncAccountsUseCase) syncMessageAttachments(
	ctx context.Context,
	tenantSlug string,
	accountID string,
	messageID string,
	attachments []domain.MailAttachmentRef,
	client domain.MailProviderClient,
) ([]domain.AttachmentRef, error) {
	if u.attachmentStorage == nil {
		return nil, nil
	}

	var refs []domain.AttachmentRef
	for _, att := range attachments {
		data, err := client.DownloadAttachment(ctx, "me", messageID, att.AttachmentID)
		if err != nil {
			return refs, fmt.Errorf("get provider attachment %s: %w", att.AttachmentID, err)
		}
		if u.maxAttachmentBytes > 0 && int64(len(data)) > u.maxAttachmentBytes {
			return refs, fmt.Errorf("attachment payload size %d exceeds max %d: %w", len(data), u.maxAttachmentBytes, errPayloadRejected)
		}

		stored, err := u.attachmentStorage.StoreAttachment(ctx, StoreAttachmentInput{
			TenantSlug:         tenantSlug,
			ConnectedAccountID: accountID,
			MessageID:          messageID,
			AttachmentID:       att.AttachmentID,
			Filename:           att.Filename,
			ContentType:        att.MimeType,
			Data:               data,
		})
		if err != nil {
			return refs, fmt.Errorf("store attachment %s: %w", att.AttachmentID, err)
		}

		refs = append(refs, domain.AttachmentRef{
			S3Key:    stored.S3Key,
			Filename: att.Filename,
			MimeType: att.MimeType,
			SHA256:   stored.SHA256,
		})
	}

	return refs, nil
}

func incrementalQuery(lastSyncedAt *time.Time) string {
	if lastSyncedAt == nil || lastSyncedAt.IsZero() {
		return ""
	}
	// Add 1 second overlap or just use exact time depending on provider
	// Most providers accept standard formats.
	return fmt.Sprintf("after:%d", lastSyncedAt.Unix())
}
