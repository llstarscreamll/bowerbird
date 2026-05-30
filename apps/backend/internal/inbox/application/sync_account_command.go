package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	connectionsApp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	platformEvents "github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type ProviderClientFactory interface {
	Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error)
}

type SyncAccountCommand struct {
	repo               domain.Repository
	connectionsService connectionsApp.InternalService
	providerFactory    ProviderClientFactory
	publisher          platformEvents.BusinessEventPublisher
	fileStore          platformStorage.FileStore
	idGenerator        func() string
	perMessageTimeout  time.Duration
	maxRawMessageBytes int
	maxAttachmentBytes int64
}

type SyncAccountCommandInput struct {
	AccountID string
}

type syncAccountRunStats struct {
	messagesSynced  int
	eventsPublished int
	failures        int
}

func NewSyncAccountCommand(
	repo domain.Repository,
	connectionsService connectionsApp.InternalService,
	providerFactory ProviderClientFactory,
	publisher platformEvents.BusinessEventPublisher,
	attachmentStore platformStorage.FileStore,
) *SyncAccountCommand {
	if repo == nil {
		panic("sync account command: repository is required")
	}

	if connectionsService == nil {
		panic("sync account command: connections service is required")
	}

	if providerFactory == nil {
		panic("sync account command: provider factory is required")
	}

	if publisher == nil {
		panic("sync account command: inbox event publisher is required")
	}

	if attachmentStore == nil {
		panic("sync account command: attachment object store is required")
	}

	return &SyncAccountCommand{
		repo:               repo,
		connectionsService: connectionsService,
		providerFactory:    providerFactory,
		publisher:          publisher,
		fileStore:          attachmentStore,
		idGenerator:        id.NewULID,
		perMessageTimeout:  60 * time.Second,
		maxRawMessageBytes: 128 * 1024 * 1024, // 128MB
		maxAttachmentBytes: 128 * 1024 * 1024, // 128MB
	}
}

func (c *SyncAccountCommand) Execute(ctx context.Context, input SyncAccountCommandInput) error {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	account, err := c.resolveActiveAccount(ctx, input.AccountID)
	if err != nil {
		return err
	}

	cursor, err := c.ensureCursor(ctx, account.ID)
	if err != nil {
		return err
	}

	stats := &syncAccountRunStats{}
	if err := c.syncAccount(ctx, tenantID, account, cursor, stats); err != nil {
		err = classifySyncError(account, err)
		stats.failures++

		now := time.Now().UTC()
		cursor.MarkSyncFailed(now, err.Error())
		_ = c.repo.UpsertSyncCursor(ctx, cursor)

		if shouldMarkRequiresReconnect(err) {
			_ = c.connectionsService.MarkRequiresReconnect(ctx, account.ID, err.Error())
		}

		return err
	}

	fmt.Printf("sync account %s completed: %+v\n", account.ID, stats)

	return nil
}

func (c *SyncAccountCommand) resolveActiveAccount(ctx context.Context, accountID string) (connectionsApp.ConnectionInfo, error) {
	if accountID == "" {
		return connectionsApp.ConnectionInfo{}, errors.New("account id is required")
	}

	accounts, err := c.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return connectionsApp.ConnectionInfo{}, fmt.Errorf("list active accounts: %w", err)
	}

	for _, account := range accounts {
		if account.ID == accountID {
			return account, nil
		}
	}

	return connectionsApp.ConnectionInfo{}, fmt.Errorf("active account not found: %s", accountID)
}

func (c *SyncAccountCommand) ensureCursor(ctx context.Context, accountID string) (*domain.InboxSyncCursor, error) {
	cursor, err := c.repo.GetSyncCursor(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if cursor == nil {
		initialSyncStart := time.Now().UTC().AddDate(0, 0, -10)
		cursor = &domain.InboxSyncCursor{
			ConnectionID: accountID,
			Status:       domain.InboxSyncStatusIdle,
			LastSyncedAt: &initialSyncStart,
		}
	}

	cursor.MarkSyncing()
	if err := c.repo.UpsertSyncCursor(ctx, cursor); err != nil {
		return nil, fmt.Errorf("upsert sync cursor: %w", err)
	}

	return cursor, nil
}

func (c *SyncAccountCommand) syncAccount(ctx context.Context, tenantID string, account connectionsApp.ConnectionInfo, cursor *domain.InboxSyncCursor, stats *syncAccountRunStats) error {
	credentialsJSON, err := c.connectionsService.DecryptCredentials(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("decrypt account credentials: %w", err)
	}

	mailClient, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	query := incrementalQuery(cursor.LastSyncedAt)
	pageToken := ""
	for {
		refs, nextPageToken, err := mailClient.ListMessages(ctx, domain.ListMessagesOptions{
			UserID:     "me",
			Query:      query,
			PageToken:  pageToken,
			MaxResults: 100,
		})
		if err != nil {
			return fmt.Errorf("list provider messages: %w", err)
		}

		for _, ref := range refs {
			if err := c.processSingleMessage(ctx, tenantID, account, ref, mailClient, stats); err != nil {
				if errors.Is(err, errPayloadRejected) {
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

	now := time.Now().UTC()
	cursor.MarkSyncSucceeded(now)
	return c.repo.UpsertSyncCursor(ctx, cursor)
}

func (c *SyncAccountCommand) processSingleMessage(
	ctx context.Context,
	tenantID string,
	account connectionsApp.ConnectionInfo,
	ref domain.MessageRef,
	client domain.MailProviderClient,
	stats *syncAccountRunStats,
) (retErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			retErr = fmt.Errorf("panic while processing provider message %s: %v: %w", ref.ID, recovered, errPayloadRejected)
		}
	}()

	messageCtx, cancel := context.WithTimeout(ctx, c.perMessageTimeout)
	defer cancel()

	message, err := client.GetMessage(messageCtx, "me", ref.ID)
	if err != nil {
		return fmt.Errorf("get provider message %s: %w", ref.ID, err)
	}

	if err := c.validateMessagePayload(message); err != nil {
		return fmt.Errorf("validate provider message %s: %w", ref.ID, err)
	}

	rawData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal provider message: %w", err)
	}
	if len(rawData) > c.maxRawMessageBytes {
		return fmt.Errorf("raw provider message size %d exceeds max %d: %w", len(rawData), c.maxRawMessageBytes, errPayloadRejected)
	}

	now := time.Now().UTC()

	emailMessage, err := domain.NewSyncedEmailMessageFromProvider(domain.NewEmailMessageFromProviderInput{
		ID:              c.idGenerator(),
		AccountID:       account.ID,
		ProviderMessage: message,
		RawData:         rawData,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return fmt.Errorf("build internal message: %w", err)
	}

	if _, err := c.repo.UpsertEmailMessage(ctx, emailMessage); err != nil {
		return fmt.Errorf("save internal message: %w", err)
	}

	var attachmentRefs []domain.AttachmentRef
	if len(message.Attachments) > 0 {
		attachmentRefs, err = c.syncMessageAttachments(messageCtx, tenantID, account.ID, message.ID, message.Attachments, client)
		if err != nil {
			return err
		}
	}

	stats.messagesSynced++

	event, err := domain.NewInboxMessageReceived(domain.NewInboxMessageReceivedInput{
		EventID:           c.idGenerator(),
		OccurredAt:        now.Format(time.RFC3339Nano),
		TenantSlug:        tenantID,
		AccountID:         account.ID,
		Provider:          account.Provider,
		ProviderMessage:   message,
		MessageInternalID: emailMessage.ID,
		AttachmentRefs:    attachmentRefs,
	})
	if err != nil {
		return fmt.Errorf("build inbox message received event: %w", err)
	}

	payload, err := domain.MarshalInboxMessageReceived(event)
	if err == nil {
		err = c.publisher.PublishBusinessEvent(ctx, platformEvents.BusinessEvent{
			Source:     domain.InboxMessageReceivedSource,
			DetailType: domain.InboxMessageReceivedDetailType,
			Detail:     payload,
		})
	}

	if err != nil {
	} else {
		stats.eventsPublished++
	}

	return nil
}

func (c *SyncAccountCommand) validateMessagePayload(message *domain.MailMessage) error {
	if err := sanitizeAndValidateMailMessage(message); err != nil {
		return fmt.Errorf("sanitize and validate message: %v: %w", err, errPayloadRejected)
	}

	if c.maxRawMessageBytes > 0 && len(message.PlainTextBody) > c.maxRawMessageBytes {
		return fmt.Errorf("plain text body exceeds max size: %w", errPayloadRejected)
	}

	for _, att := range message.Attachments {
		if c.maxAttachmentBytes > 0 && att.Size > c.maxAttachmentBytes {
			return fmt.Errorf("attachment size %d exceeds max %d: %w", att.Size, c.maxAttachmentBytes, errPayloadRejected)
		}
	}

	return nil
}

func (c *SyncAccountCommand) syncMessageAttachments(
	ctx context.Context,
	tenantID string,
	accountID string,
	messageID string,
	attachments []domain.MailAttachmentRef,
	client domain.MailProviderClient,
) ([]domain.AttachmentRef, error) {
	var refs []domain.AttachmentRef
	for _, att := range attachments {
		data, err := client.DownloadAttachment(ctx, "me", messageID, att.AttachmentID)
		if err != nil {
			return refs, fmt.Errorf("get provider attachment %s: %w", att.AttachmentID, err)
		}
		if c.maxAttachmentBytes > 0 && int64(len(data)) > c.maxAttachmentBytes {
			return refs, fmt.Errorf("attachment payload size %d exceeds max %d: %w", len(data), c.maxAttachmentBytes, errPayloadRejected)
		}

		hash := sha256.Sum256(data)
		shaHex := hex.EncodeToString(hash[:])
		objectKey := buildAttachmentObjectKey(tenantID, accountID, messageID, att.AttachmentID, att.Filename)

		_, err = c.fileStore.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
			Path:        objectKey,
			Data:        data,
			ContentType: att.MimeType,
			Metadata: map[string]string{
				"tenant_slug":          tenantID,
				"connected_account_id": accountID,
				"message_id":           messageID,
				"attachment_id":        att.AttachmentID,
				"sha256":               shaHex,
				"orig_name":            safeMetadata(att.Filename),
				"module":               "inbox",
				"stage":                "raw",
			},
		})
		if err != nil {
			return refs, fmt.Errorf("store attachment %s: %w", att.AttachmentID, err)
		}

		refs = append(refs, domain.AttachmentRef{
			S3Key:    objectKey,
			Filename: att.Filename,
			MimeType: att.MimeType,
			SHA256:   shaHex,
		})
	}

	return refs, nil
}

func buildAttachmentObjectKey(tenantID, connectedAccountID, messageID, attachmentID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf(
		"tenant/%s/inbox/%s/messages/%s/attachments/%s%s",
		tenantID,
		connectedAccountID,
		messageID,
		attachmentID,
		ext,
	)
}

func safeMetadata(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "unknown"
	}
	if len(v) > 256 {
		return v[:256]
	}
	return v
}

func incrementalQuery(lastSyncedAt *time.Time) string {
	if lastSyncedAt == nil || lastSyncedAt.IsZero() {
		return ""
	}

	return fmt.Sprintf("after:%d", lastSyncedAt.Unix())
}
