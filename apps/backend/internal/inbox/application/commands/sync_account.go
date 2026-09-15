package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	contractEvents "github.com/bowerbird/internal/contracts/events"
	inboxMappers "github.com/bowerbird/internal/inbox/application/mappers"
	"github.com/bowerbird/internal/inbox/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
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
	logger             *slog.Logger
	syncMu             sync.Mutex
	syncInFlight       map[string]struct{}
	syncCooldownUntil  map[string]time.Time
	// config
	perMessageTimeout  time.Duration
	runBudget          time.Duration
	maxMessagesPerRun  int
	maxRawMessageBytes int
	maxAttachmentBytes int64
}

const (
	defaultSyncRunBudget = 12 * time.Minute
	syncYieldReserve     = 20 * time.Second
	minWorkRemaining     = 5 * time.Second
)

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
		logger:             slog.Default(),
		syncInFlight:       map[string]struct{}{},
		syncCooldownUntil:  map[string]time.Time{},
		perMessageTimeout:  20 * time.Second,
		runBudget:          defaultSyncRunBudget,
		maxRawMessageBytes: 128 * 1024 * 1024, // 128MB
		maxAttachmentBytes: 128 * 1024 * 1024, // 128MB
	}
}

func (c *SyncAccountCommand) Execute(ctx context.Context, input SyncAccountCommandInput) error {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	if c.skipDuplicateSync(tenantID, input.AccountID) {
		return nil
	}
	defer c.endSync(tenantID, input.AccountID)

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
		c.noteRateLimitCooldown(tenantID, input.AccountID, err)

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

func syncGuardKey(tenantID, accountID string) string {
	return tenantID + ":" + accountID
}

func (c *SyncAccountCommand) skipDuplicateSync(tenantID, accountID string) bool {
	key := syncGuardKey(tenantID, accountID)
	now := time.Now()
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	if until, ok := c.syncCooldownUntil[key]; ok && now.Before(until) {
		return true
	}
	if _, busy := c.syncInFlight[key]; busy {
		return true
	}
	c.syncInFlight[key] = struct{}{}
	return false
}

func (c *SyncAccountCommand) endSync(tenantID, accountID string) {
	c.syncMu.Lock()
	delete(c.syncInFlight, syncGuardKey(tenantID, accountID))
	c.syncMu.Unlock()
}

func (c *SyncAccountCommand) noteRateLimitCooldown(tenantID, accountID string, err error) {
	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) || syncErr.Code != appErrors.CodeSyncRateLimited {
		return
	}
	retryAfter := syncErr.RetryAfterSeconds
	if retryAfter <= 0 {
		retryAfter = 120
	}
	c.syncMu.Lock()
	c.syncCooldownUntil[syncGuardKey(tenantID, accountID)] = time.Now().Add(time.Duration(retryAfter) * time.Second)
	c.syncMu.Unlock()
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
		initialSyncStart := time.Now().UTC().AddDate(0, -6, 0)
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

	deadline := c.syncDeadline(ctx, time.Now())
	complete := false
	if cursor.HistoryID() != "" {
		complete, err = c.syncAccountFromHistory(ctx, tenantID, account, cursor, mailClient, deadline)
	} else {
		complete, err = c.syncAccountFromList(ctx, tenantID, account, cursor, mailClient, deadline)
	}
	if err != nil {
		return err
	}
	if !complete {
		c.logger.Info("inbox.sync yielded",
			"tenant_id", tenantID,
			"account_id", account.ID,
			"list_page_token", cursor.ListPageToken(),
			"history_id", cursor.HistoryID(),
		)
		cursor.MarkSyncYielded()
		return c.cursorRepo.UpsertSyncCursor(ctx, cursor)
	}

	historyID, histErr := mailClient.GetHistoryID(ctx, "me")
	if histErr == nil {
		_ = cursor.AdvanceHistory(historyID)
	}

	now := time.Now().UTC()
	cursor.MarkSyncSucceeded(now)
	return c.cursorRepo.UpsertSyncCursor(ctx, cursor)
}

func (c *SyncAccountCommand) syncDeadline(ctx context.Context, started time.Time) time.Time {
	budget := c.runBudget
	if budget <= 0 {
		budget = defaultSyncRunBudget
	}
	deadline := started.Add(budget)
	if dl, ok := ctx.Deadline(); ok {
		reserved := dl.Add(-syncYieldReserve)
		if reserved.Before(deadline) {
			deadline = reserved
		}
	}
	return deadline
}

func shouldYieldSync(deadline time.Time) bool {
	return time.Until(deadline) < minWorkRemaining
}

func (c *SyncAccountCommand) processSingleMessage(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	ref domain.MessageRef,
	client domain.MailProviderClient,
	forceFull bool,
) (retErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			retErr = fmt.Errorf("panic while processing provider message %s: %v: %w", ref.ID, recovered, errPayloadRejected)
		}
	}()

	existing, err := c.messageRepo.GetInboxMessageByProviderID(ctx, account.ID, ref.ID)
	if err != nil && !errors.Is(err, domain.ErrInboxMessageNotFound) {
		return fmt.Errorf("lookup existing message %s: %w", ref.ID, err)
	}
	if errors.Is(err, domain.ErrInboxMessageNotFound) {
		existing = nil
	}
	if existing != nil && !forceFull {
		return nil
	}

	messageCtx, cancel := context.WithTimeout(ctx, c.perMessageTimeout)
	defer cancel()

	fetchedFull := forceFull
	var message *domain.MailMessage
	if forceFull {
		message, err = client.GetMessage(messageCtx, "me", ref.ID)
	} else {
		message, err = client.GetMessageMetadata(messageCtx, "me", ref.ID)
		if err != nil {
			return fmt.Errorf("get provider message metadata %s: %w", ref.ID, err)
		}
		if message.NeedsFullContent() {
			message, err = client.GetMessage(messageCtx, "me", ref.ID)
			fetchedFull = true
		}
	}
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
	priorHadFullContent := existing != nil && existing.HasFullContent()
	var inboxMessage *domain.InboxMessage
	if existing != nil {
		inboxMessage = existing
		if err := inboxMessage.ApplyProviderMessage(message, rawData, now); err != nil {
			return fmt.Errorf("apply provider message: %w", err)
		}
	} else {
		inboxMessage, err = domain.NewInboxMessageFromProvider(domain.NewInboxMessageFromProviderInput{
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
	}

	var attachmentRefs []domain.SyncedAttachmentRef
	persist := func(ctx context.Context) error {
		_, err := c.messageRepo.UpsertInboxMessage(ctx, inboxMessage)
		if err != nil {
			return fmt.Errorf("save internal message: %w", err)
		}

		if fetchedFull && len(message.Attachments) > 0 {
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

		if !fetchedFull {
			return nil
		}
		domainEvent, err := inboxMessage.NotificationAfterCapture(priorHadFullContent, domain.SyncNotificationContext{
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
			if isSkippableAttachmentError(err) {
				continue
			}
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
	deadline time.Time,
) (bool, error) {
	query := incrementalQuery(cursor.LastSyncedAt())
	pageToken := cursor.ListPageToken()
	processed := 0
	for {
		if shouldYieldSync(deadline) || c.hitMessageCap(processed) {
			cursor.CheckpointListPage(pageToken)
			if err := c.cursorRepo.UpsertSyncCursor(ctx, cursor); err != nil {
				return false, fmt.Errorf("checkpoint list page token: %w", err)
			}
			return false, nil
		}

		refs, nextPageToken, err := mailClient.ListMessages(ctx, domain.ListMessagesOptions{
			UserID:     "me",
			Query:      query,
			PageToken:  pageToken,
			MaxResults: 100,
		})
		if err != nil {
			return false, fmt.Errorf("list provider messages: %w", err)
		}

		for _, ref := range refs {
			if shouldYieldSync(deadline) || c.hitMessageCap(processed) {
				cursor.CheckpointListPage(pageToken)
				if err := c.cursorRepo.UpsertSyncCursor(ctx, cursor); err != nil {
					return false, fmt.Errorf("checkpoint list page token: %w", err)
				}
				return false, nil
			}
			if err := c.processSingleMessage(ctx, tenantID, account, ref, mailClient, false); err != nil {
				if errors.Is(err, errPayloadRejected) {
					processed++
					continue
				}

				return false, err
			}
			processed++
		}

		cursor.CheckpointListPage(nextPageToken)
		if err := c.cursorRepo.UpsertSyncCursor(ctx, cursor); err != nil {
			return false, fmt.Errorf("checkpoint list page token: %w", err)
		}

		pageToken = nextPageToken
		if pageToken == "" {
			return true, nil
		}
	}
}

func (c *SyncAccountCommand) syncAccountFromHistory(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	cursor *domain.SyncCursor,
	mailClient domain.MailProviderClient,
	deadline time.Time,
) (bool, error) {
	page, err := mailClient.ListHistory(ctx, "me", cursor.HistoryID())
	if err != nil {
		return false, fmt.Errorf("list provider history: %w", err)
	}
	if page.Expired {
		return c.syncAccountFromList(ctx, tenantID, account, cursor, mailClient, deadline)
	}

	processed := 0
	for _, change := range page.Changes {
		if shouldYieldSync(deadline) || c.hitMessageCap(processed) {
			return false, nil
		}
		if change.Type == domain.HistoryChangeDeleted {
			continue
		}
		if change.Type == domain.HistoryChangeUpdated {
			if err := c.refreshExistingMessage(ctx, tenantID, account, change.MessageID, mailClient); err != nil {
				if errors.Is(err, errPayloadRejected) {
					processed++
					continue
				}
				return false, err
			}
			processed++
			continue
		}
		if err := c.processSingleMessage(ctx, tenantID, account, domain.MessageRef{ID: change.MessageID}, mailClient, false); err != nil {
			if errors.Is(err, errPayloadRejected) {
				processed++
				continue
			}
			return false, err
		}
		processed++
	}

	if page.NewHistoryID != "" {
		_ = cursor.AdvanceHistory(page.NewHistoryID)
	}

	return true, nil
}

func (c *SyncAccountCommand) hitMessageCap(processed int) bool {
	return c.maxMessagesPerRun > 0 && processed >= c.maxMessagesPerRun
}

func (c *SyncAccountCommand) refreshExistingMessage(
	ctx context.Context,
	tenantID string,
	account connectionsapi.ConnectionInfo,
	providerMessageID string,
	client domain.MailProviderClient,
) error {
	existing, err := c.messageRepo.GetInboxMessageByProviderID(ctx, account.ID, providerMessageID)
	if err != nil {
		if errors.Is(err, domain.ErrInboxMessageNotFound) {
			return c.processSingleMessage(ctx, tenantID, account, domain.MessageRef{ID: providerMessageID}, client, false)
		}
		return err
	}

	meta, err := client.GetMessageMetadata(ctx, "me", providerMessageID)
	if err != nil {
		return fmt.Errorf("get provider message metadata %s: %w", providerMessageID, err)
	}
	existing.ApplyProviderFlags(meta.LabelIDs, time.Now().UTC())
	if err := c.messageRepo.UpdateInboxMessageFlags(ctx, existing); err != nil {
		return err
	}
	if existing.HasFullContent() || !meta.NeedsFullContent() {
		return nil
	}
	return c.processSingleMessage(ctx, tenantID, account, domain.MessageRef{ID: providerMessageID}, client, true)
}

func (c *SyncAccountCommand) hydrateMessage(ctx context.Context, messageID string) error {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}
	existing, err := c.messageRepo.GetInboxMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if existing.HasFullContent() {
		return nil
	}

	account, err := c.connectionsService.GetConnection(ctx, existing.ConnectionID())
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	credentialsJSON, err := c.connectionsService.DecryptCredentials(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("decrypt account credentials: %w", err)
	}
	mailClient, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}
	return c.processSingleMessage(ctx, tenantID, account, domain.MessageRef{ID: existing.ProviderMessageID()}, mailClient, true)
}

func LimitMessagesPerRun(c *SyncAccountCommand, n int) {
	if c == nil {
		return
	}
	if n < 0 {
		n = 0
	}
	c.maxMessagesPerRun = n
}
