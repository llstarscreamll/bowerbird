package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	contractEvents "github.com/bowerbird/internal/contracts/events"
	inboxMappers "github.com/bowerbird/internal/inbox/application/mappers"
	"github.com/bowerbird/internal/inbox/domain"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/id"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
)

type ProviderClientFactory interface {
	Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error)
}

type UnitOfWorkRunner interface {
	Run(ctx context.Context, fn func(context.Context) error) error
}

type SyncAccountCommand struct {
	cursorRepo         domain.SyncCursorRepository
	messageRepo        domain.MessageRepository
	connectionsService connectionsapi.InternalService
	providerFactory    ProviderClientFactory
	eventBus           platformEvents.EventBus
	fileStore          platformStorage.FileStore
	unitOfWork         UnitOfWorkRunner
	idGenerator        func() string
	// config
	perMessageTimeout  time.Duration
	maxRawMessageBytes int
	maxAttachmentBytes int64
}

type SyncAccountCommandInput struct {
	AccountID string
}

func NewSyncAccountCommand(
	cursorRepo domain.SyncCursorRepository,
	messageRepo domain.MessageRepository,
	connectionsService connectionsapi.InternalService,
	providerFactory ProviderClientFactory,
	eventBus platformEvents.EventBus,
	fileStore platformStorage.FileStore,
	unitOfWork UnitOfWorkRunner,
) *SyncAccountCommand {
	if cursorRepo == nil {
		panic("sync account command: sync cursor repository is required")
	}

	if messageRepo == nil {
		panic("sync account command: message repository is required")
	}

	if connectionsService == nil {
		panic("sync account command: connections service is required")
	}

	if providerFactory == nil {
		panic("sync account command: provider factory is required")
	}

	if eventBus == nil {
		panic("sync account command: inbox event publisher is required")
	}

	if fileStore == nil {
		panic("sync account command: attachment object store is required")
	}

	if unitOfWork == nil {
		panic("sync account command: unit of work is required")
	}

	return &SyncAccountCommand{
		cursorRepo:         cursorRepo,
		messageRepo:        messageRepo,
		connectionsService: connectionsService,
		providerFactory:    providerFactory,
		eventBus:           eventBus,
		fileStore:          fileStore,
		unitOfWork:         unitOfWork,
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

	if err := c.syncAccount(ctx, tenantID, account, cursor); err != nil {
		err = classifySyncError(account, err)

		cursor.MarkSyncFailed(err.Error())
		if persistErr := c.cursorRepo.UpsertSyncCursor(ctx, cursor); persistErr != nil {
			return fmt.Errorf("%w; persist sync cursor: %v", err, persistErr)
		}

		if shouldMarkRequiresReconnect(err) {
			if markErr := c.connectionsService.MarkRequiresReconnect(ctx, account.ID, err.Error()); markErr != nil {
				return fmt.Errorf("%w; mark requires reconnect: %v", err, markErr)
			}
		}

		return err
	}

	return nil
}

func (c *SyncAccountCommand) resolveActiveAccount(ctx context.Context, accountID string) (connectionsapi.ConnectionInfo, error) {
	if accountID == "" {
		return connectionsapi.ConnectionInfo{}, errors.New("account id is required")
	}

	accounts, err := c.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return connectionsapi.ConnectionInfo{}, fmt.Errorf("list active accounts: %w", err)
	}

	for _, account := range accounts {
		if account.ID == accountID {
			return account, nil
		}
	}

	return connectionsapi.ConnectionInfo{}, fmt.Errorf("active account not found: %s", accountID)
}

func (c *SyncAccountCommand) ensureCursor(ctx context.Context, accountID string) (*domain.SyncCursor, error) {
	cursor, err := c.cursorRepo.GetSyncCursor(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if cursor == nil {
		initialSyncStart := time.Now().UTC().AddDate(0, -2, 0)
		cursor, err = domain.NewSyncCursor(accountID, &initialSyncStart)
		if err != nil {
			return nil, fmt.Errorf("new sync cursor: %w", err)
		}
	}

	cursor.MarkSyncing()
	if err := c.cursorRepo.UpsertSyncCursor(ctx, cursor); err != nil {
		return nil, fmt.Errorf("upsert sync cursor: %w", err)
	}

	return cursor, nil
}

func (c *SyncAccountCommand) syncAccount(ctx context.Context, tenantID string, account connectionsapi.ConnectionInfo, cursor *domain.SyncCursor) error {
	credentialsJSON, err := c.connectionsService.DecryptCredentials(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("decrypt account credentials: %w", err)
	}

	mailClient, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	if cursor.HistoryID() != "" {
		if err := c.syncAccountFromHistory(ctx, tenantID, account, cursor, mailClient); err != nil {
			return err
		}
	} else {
		if err := c.syncAccountFromList(ctx, tenantID, account, cursor, mailClient); err != nil {
			return err
		}
	}

	historyID, histErr := mailClient.GetHistoryID(ctx, "me")
	if histErr == nil {
		_ = cursor.AdvanceHistory(historyID)
	}

	now := time.Now().UTC()
	cursor.MarkSyncSucceeded(now)
	return c.cursorRepo.UpsertSyncCursor(ctx, cursor)
}

func (c *SyncAccountCommand) processSingleMessage(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	ref domain.MessageRef,
	client domain.MailProviderClient,
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

	inboxMessage, err := domain.NewInboxMessageFromProvider(domain.NewInboxMessageFromProviderInput{
		ID:              c.idGenerator(),
		ConnectionID:    account.ID,
		ProviderMessage: message,
		RawData:         rawData,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return fmt.Errorf("build internal message: %w", err)
	}

	var attachmentRefs []domain.SyncedAttachmentRef
	persist := func(ctx context.Context) error {
		var err error
		inserted, err := c.messageRepo.UpsertInboxMessage(ctx, inboxMessage)
		if err != nil {
			return fmt.Errorf("save internal message: %w", err)
		}

		if len(message.Attachments) > 0 {
			attachmentRefs, err = c.syncMessageAttachments(
				ctx,
				messageCtx,
				tenantID,
				account.ID,
				inboxMessage.ID(),
				message.ID,
				message.Attachments,
				client,
			)
			if err != nil {
				return err
			}
		}

		if !inserted {
			return nil
		}
		domainEvent, err := inboxMessage.NotificationAfterPersist(inserted, domain.SyncNotificationContext{
			EventID:         c.idGenerator(),
			TenantSlug:      tenantID,
			AccountID:       account.ID,
			Provider:        account.Provider,
			ProviderMessage: message,
			AttachmentRefs:  attachmentRefs,
		})
		if err != nil {
			return fmt.Errorf("build message synced event: %w", err)
		}
		if domainEvent == nil {
			return nil
		}
		if err := c.publishMessageSynced(ctx, *domainEvent); err != nil {
			return fmt.Errorf("publish inbox message received event: %w", err)
		}
		return nil
	}

	return c.unitOfWork.Run(ctx, persist)
}

func (c *SyncAccountCommand) publishMessageSynced(ctx context.Context, event domain.MessageSynced) error {
	payload, err := inboxMappers.MarshalMessageSyncedPayload(event)
	if err != nil {
		return fmt.Errorf("marshal inbox message received event: %w", err)
	}

	err = c.eventBus.Publish(ctx, platformEvents.BusinessEvent{
		Source:     contractEvents.InboxMessageReceivedSource,
		DetailType: contractEvents.InboxMessageReceivedDetailType,
		Detail:     payload,
	})
	if err != nil {
		return fmt.Errorf("publish inbox message received event: %w", err)
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
	dbCtx context.Context,
	providerCtx context.Context,
	tenantID string,
	connectionID string,
	inboxMessageID string,
	providerMessageID string,
	attachments []domain.MailAttachmentRef,
	client domain.MailProviderClient,
) ([]domain.SyncedAttachmentRef, error) {
	var refs []domain.SyncedAttachmentRef
	now := time.Now().UTC()
	for _, att := range attachments {
		data, err := client.DownloadAttachment(providerCtx, "me", providerMessageID, att.AttachmentID)
		if err != nil {
			return refs, fmt.Errorf("get provider attachment %s: %w", att.AttachmentID, err)
		}
		if c.maxAttachmentBytes > 0 && int64(len(data)) > c.maxAttachmentBytes {
			return refs, fmt.Errorf("attachment payload size %d exceeds max %d: %w", len(data), c.maxAttachmentBytes, errPayloadRejected)
		}

		hash := sha256.Sum256(data)
		shaHex := hex.EncodeToString(hash[:])

		existing, err := c.messageRepo.GetMessageAttachmentByMessageAndSHA(dbCtx, inboxMessageID, shaHex)
		if err != nil {
			return refs, fmt.Errorf("lookup attachment %s: %w", att.AttachmentID, err)
		}
		if existing != nil {
			refs = append(refs, domain.SyncedAttachmentRef{
				S3Key:    existing.S3Key,
				Filename: existing.Filename,
				MimeType: derefString(existing.MimeType),
				SHA256:   existing.SHA256,
			})
			continue
		}

		storageFileID := c.idGenerator()
		objectKey := platformStorage.InboxAttachmentObjectKey(tenantID, connectionID, inboxMessageID, storageFileID, att.Filename)

		_, err = c.fileStore.WriteFileIfAbsent(dbCtx, platformStorage.WriteFileIfAbsentInput{
			Path:        objectKey,
			Data:        data,
			ContentType: att.MimeType,
			Metadata: map[string]string{
				"tenant_id":           tenantID,
				"connection_id":       connectionID,
				"provider_message_id": providerMessageID,
				"message_id":          inboxMessageID,
				"sha256":              shaHex,
				"orig_name":           att.Filename,
				"module":              "inbox",
				"stage":               "raw",
			},
		})
		if err != nil {
			return refs, fmt.Errorf("store attachment %s: %w", att.AttachmentID, err)
		}

		sizeBytes := int64(len(data))
		attachment, err := domain.NewMessageAttachment(domain.NewMessageAttachmentInput{
			ID:        storageFileID,
			MessageID: inboxMessageID,
			Filename:  att.Filename,
			MimeType:  pointerIfNotEmpty(att.MimeType),
			SizeBytes: &sizeBytes,
			SHA256:    shaHex,
			S3Key:     objectKey,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return refs, fmt.Errorf("build message attachment %s: %w", att.AttachmentID, err)
		}

		if _, err := c.messageRepo.UpsertMessageAttachment(dbCtx, attachment); err != nil {
			return refs, fmt.Errorf("save message attachment %s: %w", att.AttachmentID, err)
		}

		refs = append(refs, domain.SyncedAttachmentRef{
			S3Key:    objectKey,
			Filename: att.Filename,
			MimeType: att.MimeType,
			SHA256:   shaHex,
		})
	}

	return refs, nil
}

func pointerIfNotEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	v := value
	return &v
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func incrementalQuery(lastSyncedAt *time.Time) string {
	if lastSyncedAt == nil || lastSyncedAt.IsZero() {
		return ""
	}

	return fmt.Sprintf("after:%d", lastSyncedAt.Unix())
}

func (c *SyncAccountCommand) syncAccountFromList(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	cursor *domain.SyncCursor,
	mailClient domain.MailProviderClient,
) error {
	query := incrementalQuery(cursor.LastSyncedAt())
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
			if err := c.processSingleMessage(ctx, tenantID, account, ref, mailClient); err != nil {
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

	return nil
}

func (c *SyncAccountCommand) syncAccountFromHistory(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	cursor *domain.SyncCursor,
	mailClient domain.MailProviderClient,
) error {
	page, err := mailClient.ListHistory(ctx, "me", cursor.HistoryID())
	if err != nil {
		return fmt.Errorf("list provider history: %w", err)
	}
	if page.Expired {
		return c.syncAccountFromList(ctx, tenantID, account, cursor, mailClient)
	}

	for _, change := range page.Changes {
		if change.Type == domain.HistoryChangeDeleted {
			continue
		}
		if err := c.processSingleMessage(ctx, tenantID, account, domain.MessageRef{ID: change.MessageID}, mailClient); err != nil {
			if errors.Is(err, errPayloadRejected) {
				continue
			}
			return err
		}
	}

	if page.NewHistoryID != "" {
		_ = cursor.AdvanceHistory(page.NewHistoryID)
	}

	return nil
}
