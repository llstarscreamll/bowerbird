package application_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	connectionsApp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	inboxApp "github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	platformEvents "github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAccountCommand_RequiresAccountID(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "account id is required")
}

func TestSyncAccountCommand_FailsWhenAccountIsNotActive(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-2", Provider: "gmail", ProviderAccountEmail: "other@gmail.com"}},
	}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "active account not found: acc-1")
	assert.Empty(t, providerClient.listQueries)
}

func TestSyncAccountCommand_CreatesCursorForLastTenDaysWhenMissing(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, providerClient.listQueries, 1)

	query := providerClient.listQueries[0]
	require.True(t, strings.HasPrefix(query, "after:"))
	queryTs, convErr := strconv.ParseInt(strings.TrimPrefix(query, "after:"), 10, 64)
	require.NoError(t, convErr)

	expected := time.Now().UTC().AddDate(0, 0, -10)
	assert.WithinDuration(t, expected, time.Unix(queryTs, 0).UTC(), 5*time.Second)

	require.Len(t, repo.upsertedCursors, 2)
	assert.Equal(t, domain.InboxSyncStatusSyncing, repo.upsertedCursors[0].Status)
	assert.Equal(t, domain.InboxSyncStatusIdle, repo.upsertedCursors[1].Status)
}

func TestSyncAccountCommand_UsesExistingCursorWithoutResettingRange(t *testing.T) {
	previousSync := time.Date(2026, 5, 2, 8, 30, 0, 0, time.UTC)
	repo := newFakeInboxRepo()
	repo.cursors["acc-1"] = &domain.InboxSyncCursor{
		ConnectionID: "acc-1",
		LastSyncedAt: &previousSync,
		Status:       domain.InboxSyncStatusIdle,
	}

	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}
	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)

	expectedQuery := "after:" + toUnixString(previousSync)
	require.NotEmpty(t, providerClient.listQueries)
	assert.Equal(t, expectedQuery, providerClient.listQueries[0])
}

func TestSyncAccountCommand_ContinuesAfterPayloadRejected(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{
		refs: []domain.MessageRef{{ID: "m-invalid"}, {ID: "m-valid"}},
		messages: map[string]*domain.MailMessage{
			"m-invalid": {
				ID:            "m-invalid",
				Sender:        `<img src=x onerror=alert(1)>`,
				PlainTextBody: "bad sender",
			},
			"m-valid": {
				ID:            "m-valid",
				ThreadID:      "t-2",
				Subject:       "ok",
				Sender:        "Sender <sender@example.com>",
				PlainTextBody: "normal",
			},
		},
	}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, repo.upsertedMessages, 1)
	assert.Equal(t, []string{"m-invalid", "m-valid"}, providerClient.getMessageCalls)

	persisted := repo.upsertedMessages[0]
	require.NotNil(t, persisted.SenderEmail)
	assert.Equal(t, "sender@example.com", *persisted.SenderEmail)
}

func TestSyncAccountCommand_UsesProviderMessageIDForAttachmentDownload(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{
		refs: []domain.MessageRef{{ID: "provider-msg-1"}},
		messages: map[string]*domain.MailMessage{
			"provider-msg-1": {
				ID:            "provider-msg-1",
				ThreadID:      "thread-1",
				Subject:       "with attachment",
				Sender:        "Sender <sender@example.com>",
				PlainTextBody: "normal",
				Attachments: []domain.MailAttachmentRef{
					{AttachmentID: "att-1", Filename: "doc.xml", MimeType: "application/xml", Size: 10},
				},
			},
		},
	}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, providerClient.downloadAttachmentCalls, 1)
	assert.Equal(t, "provider-msg-1", providerClient.downloadAttachmentCalls[0].messageID)
	assert.Equal(t, "att-1", providerClient.downloadAttachmentCalls[0].attachmentID)
}

func TestSyncAccountCommand_FailsWhenAttachmentDownloadFails(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{
		refs: []domain.MessageRef{{ID: "provider-msg-1"}},
		messages: map[string]*domain.MailMessage{
			"provider-msg-1": {
				ID:            "provider-msg-1",
				ThreadID:      "thread-1",
				Subject:       "with attachment",
				Sender:        "Sender <sender@example.com>",
				PlainTextBody: "normal",
				Attachments: []domain.MailAttachmentRef{
					{AttachmentID: "att-1", Filename: "doc.xml", MimeType: "application/xml", Size: 10},
				},
			},
		},
		downloadAttachmentErr: errors.New("attachment api unavailable"),
	}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "get provider attachment att-1")
}

func TestSyncAccountCommand_ReauthMarksReconnect(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{listErr: errors.New("list failed with status 401")}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxApp.NewSyncAccountCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxApp.SyncAccountCommandInput{AccountID: "acc-1"})
	require.Error(t, err)
	assert.Equal(t, 1, connectionsSvc.markReconnectCalls)

	cursor := repo.cursors["acc-1"]
	require.NotNil(t, cursor)
	assert.Equal(t, domain.InboxSyncStatusError, cursor.Status)
}

func toUnixString(v time.Time) string {
	return strconv.FormatInt(v.Unix(), 10)
}

type fakeInboxRepo struct {
	cursors          map[string]*domain.InboxSyncCursor
	upsertedCursors  []*domain.InboxSyncCursor
	upsertedMessages []*domain.EmailMessage
}

func newFakeInboxRepo() *fakeInboxRepo {
	return &fakeInboxRepo{cursors: map[string]*domain.InboxSyncCursor{}}
}

func (f *fakeInboxRepo) GetSyncCursor(ctx context.Context, connectionID string) (*domain.InboxSyncCursor, error) {
	return f.cursors[connectionID], nil
}

func (f *fakeInboxRepo) UpsertSyncCursor(ctx context.Context, cursor *domain.InboxSyncCursor) error {
	cloned := *cursor
	f.cursors[cursor.ConnectionID] = &cloned
	f.upsertedCursors = append(f.upsertedCursors, &cloned)
	return nil
}

func (f *fakeInboxRepo) UpsertEmailMessage(ctx context.Context, msg *domain.EmailMessage) (bool, error) {
	f.upsertedMessages = append(f.upsertedMessages, msg)
	return true, nil
}

func (f *fakeInboxRepo) UpsertEmailAttachment(ctx context.Context, attachment *domain.EmailAttachment) (bool, error) {
	return true, nil
}

func (f *fakeInboxRepo) ListUnifiedMessages(ctx context.Context) ([]domain.UnifiedMessage, error) {
	return nil, nil
}

func (f *fakeInboxRepo) ListMessagesByAccount(ctx context.Context, accountID string, limit, offset int) ([]domain.UnifiedMessage, error) {
	return nil, nil
}

func (f *fakeInboxRepo) GetMessageByID(ctx context.Context, messageID string) (*domain.UnifiedMessage, error) {
	return nil, nil
}

func (f *fakeInboxRepo) GetMessageAttachments(ctx context.Context, messageID string) ([]domain.EmailAttachment, error) {
	return nil, nil
}

type fakeConnectionsInternalService struct {
	activeConnections  []connectionsApp.ConnectionInfo
	markReconnectCalls int
}

func (f *fakeConnectionsInternalService) GetActiveConnections(ctx context.Context) ([]connectionsApp.ConnectionInfo, error) {
	return f.activeConnections, nil
}

func (f *fakeConnectionsInternalService) DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error) {
	return []byte(`{"access_token":"masked"}`), nil
}

func (f *fakeConnectionsInternalService) MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error {
	f.markReconnectCalls++
	return nil
}

func (f *fakeConnectionsInternalService) GetSharingPolicy(ctx context.Context, connectionID string) (string, error) {
	return "private", nil
}

type fakeProviderFactory struct {
	client domain.MailProviderClient
	err    error
}

func (f *fakeProviderFactory) Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

type fakeProviderClient struct {
	refs                    []domain.MessageRef
	messages                map[string]*domain.MailMessage
	listErr                 error
	listQueries             []string
	getMessageCalls         []string
	downloadAttachmentCalls []attachmentDownloadCall
	downloadAttachmentErr   error
}

func (f *fakeProviderClient) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	f.listQueries = append(f.listQueries, opts.Query)
	if f.listErr != nil {
		return nil, "", f.listErr
	}
	return f.refs, "", nil
}

func (f *fakeProviderClient) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	f.getMessageCalls = append(f.getMessageCalls, messageID)
	message, ok := f.messages[messageID]
	if !ok {
		return nil, errors.New("message not found")
	}
	return message, nil
}

func (f *fakeProviderClient) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
	f.downloadAttachmentCalls = append(f.downloadAttachmentCalls, attachmentDownloadCall{
		messageID:    messageID,
		attachmentID: attachmentID,
	})
	if f.downloadAttachmentErr != nil {
		return nil, f.downloadAttachmentErr
	}
	return []byte("attachment"), nil
}

func (f *fakeProviderClient) DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []domain.MailAttachmentRef) ([]domain.DownloadedMailAttachment, error) {
	return nil, nil
}

func (f *fakeProviderClient) CreateLabel(ctx context.Context, userID, labelName string) (string, error) {
	return "", nil
}

func (f *fakeProviderClient) AddLabelToMessage(ctx context.Context, userID, messageID, labelID string) error {
	return nil
}

type fakeInboxEventPublisher struct {
	published []platformEvents.BusinessEvent
}

func (f *fakeInboxEventPublisher) PublishBusinessEvent(ctx context.Context, event platformEvents.BusinessEvent) error {
	f.published = append(f.published, event)
	return nil
}

type fakeFileStore struct {
	inputs []platformStorage.WriteFileIfAbsentInput
}

func (f *fakeFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	f.inputs = append(f.inputs, input)
	return &platformStorage.WriteFileIfAbsentResult{
		Written:   true,
		SizeBytes: int64(len(input.Data)),
	}, nil
}

func (f *fakeFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}

type attachmentDownloadCall struct {
	messageID    string
	attachmentID string
}

var _ connectionsApp.InternalService = (*fakeConnectionsInternalService)(nil)
var _ domain.Repository = (*fakeInboxRepo)(nil)
var _ domain.MailProviderClient = (*fakeProviderClient)(nil)
var _ platformEvents.BusinessEventPublisher = (*fakeInboxEventPublisher)(nil)
var _ platformStorage.FileStore = (*fakeFileStore)(nil)
