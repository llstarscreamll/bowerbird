package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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
	credentialsService *CredentialsService
	providerFactory    ProviderClientFactory
	publisher          InboxEventPublisher
	attachmentStorage  AttachmentStorage
	logger             *slog.Logger
	metrics            observability.Metrics
	now                func() time.Time
	newID              func() string
}

type SyncAccountsResult struct {
	AccountsProcessed int
	MessagesSynced    int
	EventsPublished   int
	Failures          int
}

func NewSyncAccountsUseCase(
	repo domain.Repository,
	credentialsService *CredentialsService,
	providerFactory ProviderClientFactory,
	publisher InboxEventPublisher,
	attachmentStorage AttachmentStorage,
) *SyncAccountsUseCase {
	return &SyncAccountsUseCase{
		repo:               repo,
		credentialsService: credentialsService,
		providerFactory:    providerFactory,
		publisher:          publisher,
		attachmentStorage:  attachmentStorage,
		logger:             slog.Default(),
		metrics:            observability.NoopMetrics{},
		now:                time.Now,
		newID:              id.NewULID,
	}
}

func (u *SyncAccountsUseCase) Run(ctx context.Context) (*SyncAccountsResult, error) {
	startedAt := u.now()
	tenantSlug, err := tenant.TenantSlugFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := u.repo.ListConnectedAccountsByStatus(ctx, "active")
	if err != nil {
		return nil, fmt.Errorf("list active connected accounts: %w", err)
	}

	result := &SyncAccountsResult{}
	for _, account := range accounts {
		result.AccountsProcessed++
		if err := u.syncAccount(ctx, tenantSlug, account, result); err != nil {
			result.Failures++
			u.metrics.IncCounter("inbox_sync_errors_total", map[string]string{"tenant_slug": tenantSlug, "provider": account.Provider})
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
			if markErr := account.MarkSyncFailed(now, err.Error()); markErr != nil {
				u.logger.Error("inbox account state transition failed", "account_id", account.ID, "error", markErr)
			} else {
				_ = u.repo.UpdateConnectedAccountSyncState(ctx, account.ID, account.Status, account.LastSyncedAt, account.LastError, account.UpdatedAt)
			}
			continue
		}

		now := u.now().UTC()
		if err := account.MarkSyncSucceeded(now); err != nil {
			u.logger.Error("inbox account state transition failed", "account_id", account.ID, "error", err)
		} else {
			_ = u.repo.UpdateConnectedAccountSyncState(ctx, account.ID, account.Status, account.LastSyncedAt, account.LastError, account.UpdatedAt)
		}
	}

	u.metrics.ObserveDuration("inbox_sync_run_duration_ms", u.now().Sub(startedAt), map[string]string{"tenant_slug": tenantSlug})
	u.logger.Info(
		"inbox sync run completed",
		"tenant_slug",
		tenantSlug,
		"accounts_processed",
		result.AccountsProcessed,
		"messages_synced",
		result.MessagesSynced,
		"events_published",
		result.EventsPublished,
		"failures",
		result.Failures,
	)

	return result, nil
}

func (u *SyncAccountsUseCase) syncAccount(ctx context.Context, tenantSlug string, account *domain.ConnectedAccount, result *SyncAccountsResult) error {
	credentialsJSON, err := u.credentialsService.DecryptFromStorage(account.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("decrypt account credentials: %w", err)
	}

	client, err := u.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	query := incrementalQuery(account.LastSyncedAt)
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
			message, err := client.GetMessage(ctx, "me", ref.ID)
			if err != nil {
				return fmt.Errorf("get provider message %s: %w", ref.ID, err)
			}

			rawData, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("marshal provider message: %w", err)
			}

			now := u.now().UTC()
			emailMessage, err := domain.NewSyncedEmailMessage(domain.NewEmailMessageInput{
				ID:                u.newID(),
				AccountID:         account.ID,
				ProviderMessageID: message.ID,
				ProviderThreadID:  asPtr(message.ThreadID),
				Subject:           asPtr(message.Subject),
				SenderEmail:       asPtr(message.Sender),
				ReceivedAt:        message.ReceivedAt,
				RawData:           rawData,
				CreatedAt:         now,
				UpdatedAt:         now,
			})
			if err != nil {
				return fmt.Errorf("build synced email message: %w", err)
			}

			inserted, err := u.repo.UpsertEmailMessage(ctx, emailMessage)
			if err != nil {
				return fmt.Errorf("upsert email message: %w", err)
			}

			if inserted {
				result.MessagesSynced++
				u.metrics.IncCounter("inbox_sync_messages_total", map[string]string{"tenant_slug": tenantSlug, "provider": account.Provider})

				eventAttachmentRefs, err := u.syncMessageAttachments(ctx, tenantSlug, emailMessage, message, client)
				if err != nil {
					return fmt.Errorf("sync message attachments: %w", err)
				}

				event := domain.InboxMessageReceived{
					EventID:           u.newID(),
					OccurredAt:        now.Format(time.RFC3339),
					TenantSlug:        tenantSlug,
					AccountID:         account.ID,
					Provider:          account.Provider,
					ProviderMessageID: message.ID,
					MessageInternalID: emailMessage.ID,
					Subject:           message.Subject,
					Sender:            message.Sender,
					AttachmentRefs:    eventAttachmentRefs,
					RawDataRef:        emailMessage.ID,
				}

				if message.ReceivedAt != nil {
					event.ReceivedAt = message.ReceivedAt.Format(time.RFC3339)
				}

				if err := u.publisher.PublishInboxMessageReceived(ctx, event); err != nil {
					return fmt.Errorf("publish InboxMessageReceived event: %w", err)
				}

				result.EventsPublished++
				u.logger.Info(
					"inbox message synced and event published",
					"tenant_slug",
					tenantSlug,
					"account_id",
					account.ID,
					"provider",
					account.Provider,
					"provider_message_id",
					message.ID,
					"attachments",
					len(eventAttachmentRefs),
				)
			} else {
				u.logger.Info(
					"inbox message already upserted, skipped event publish",
					"tenant_slug",
					tenantSlug,
					"account_id",
					account.ID,
					"provider_message_id",
					message.ID,
				)
			}
		}

		if nextPageToken == "" {
			break
		}

		pageToken = nextPageToken
	}

	return nil
}

func incrementalQuery(lastSyncedAt *time.Time) string {
	if lastSyncedAt == nil {
		return ""
	}

	return fmt.Sprintf("after:%d", lastSyncedAt.UTC().Unix())
}

func asPtr(value string) *string {
	if value == "" {
		return nil
	}

	v := value
	return &v
}

func (u *SyncAccountsUseCase) syncMessageAttachments(
	ctx context.Context,
	tenantSlug string,
	emailMessage *domain.EmailMessage,
	providerMessage *domain.MailMessage,
	client domain.MailProviderClient,
) ([]domain.AttachmentRef, error) {
	if len(providerMessage.Attachments) == 0 {
		return nil, nil
	}

	if u.attachmentStorage == nil {
		return nil, fmt.Errorf("attachment storage is not configured")
	}

	downloaded, err := client.DownloadMessageAttachments(ctx, "me", providerMessage.ID, providerMessage.Attachments)
	if err != nil {
		return nil, fmt.Errorf("download message attachments: %w", err)
	}

	eventRefs := make([]domain.AttachmentRef, 0, len(downloaded))
	now := u.now().UTC()
	for _, item := range downloaded {
		attachmentRecordID := u.newID()

		stored, err := u.attachmentStorage.StoreAttachment(ctx, StoreAttachmentInput{
			TenantSlug:         tenantSlug,
			ConnectedAccountID: emailMessage.AccountID,
			MessageID:          emailMessage.ID,
			AttachmentID:       attachmentRecordID,
			Filename:           item.Filename,
			ContentType:        item.MimeType,
			Data:               item.Data,
		})
		if err != nil {
			return nil, fmt.Errorf("store attachment %s: %w", item.Filename, err)
		}

		sizeBytes := stored.SizeBytes
		rawAttachmentData, err := json.Marshal(map[string]any{
			"uploaded":      true,
			"attachment_id": item.AttachmentID,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal attachment raw data: %w", err)
		}

		attachment, err := domain.NewEmailAttachment(domain.NewEmailAttachmentInput{
			ID:        attachmentRecordID,
			MessageID: emailMessage.ID,
			Filename:  item.Filename,
			MimeType:  asPtr(item.MimeType),
			SizeBytes: &sizeBytes,
			SHA256:    stored.SHA256,
			S3Key:     stored.S3Key,
			RawData:   rawAttachmentData,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return nil, fmt.Errorf("build email attachment: %w", err)
		}

		if _, err := u.repo.UpsertEmailAttachment(ctx, attachment); err != nil {
			return nil, fmt.Errorf("upsert email attachment: %w", err)
		}

		eventRefs = append(eventRefs, domain.AttachmentRef{
			S3Key:    stored.S3Key,
			Filename: item.Filename,
			MimeType: item.MimeType,
			SHA256:   stored.SHA256,
		})
	}

	return eventRefs, nil
}
