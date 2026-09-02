package application_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	connectionsApp "github.com/bowerbird/internal/connections/application"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/domain"
	platformEvents "github.com/bowerbird/internal/platform/events"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUnitOfWork struct{}

func (fakeUnitOfWork) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestSyncAccountCommand_RequiresAccountID(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{})
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, providerClient.listQueries, 1)

	query := providerClient.listQueries[0]
	require.True(t, strings.HasPrefix(query, "after:"))
	queryTs, convErr := strconv.ParseInt(strings.TrimPrefix(query, "after:"), 10, 64)
	require.NoError(t, convErr)

	expected := time.Now().UTC().AddDate(0, -2, 0)
	assert.WithinDuration(t, expected, time.Unix(queryTs, 0).UTC(), 5*time.Second)

	require.Len(t, repo.upsertedCursors, 2)
	assert.Equal(t, domain.SyncCursorStatusSyncing, repo.upsertedCursors[0].Status())
	assert.Equal(t, domain.SyncCursorStatusIdle, repo.upsertedCursors[1].Status())
}

func TestSyncAccountCommand_UsesExistingCursorWithoutResettingRange(t *testing.T) {
	previousSync := time.Date(2026, 5, 2, 8, 30, 0, 0, time.UTC)
	repo := newFakeInboxRepo()
	repo.cursors["acc-1"] = domain.RehydrateSyncCursor(domain.SyncCursorSnapshot{
		ConnectionID: "acc-1",
		LastSyncedAt: &previousSync,
		Status:       domain.SyncCursorStatusIdle,
	})

	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{}
	publisher := &fakeInboxEventPublisher{}
	attachmentStore := &fakeFileStore{}
	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, repo.upsertedMessages, 1)
	assert.Equal(t, []string{"m-invalid", "m-valid"}, providerClient.getMessageCalls)

	persisted := repo.upsertedMessages[0]
	require.NotNil(t, persisted.SenderEmail())
	assert.Equal(t, "sender@example.com", *persisted.SenderEmail())
}

func TestSyncAccountCommand_DoesNotRepublishExistingMessage(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	mail := &domain.MailMessage{
		ID:            "m-1",
		ThreadID:      "t-1",
		Subject:       "ok",
		Sender:        "sender@example.com",
		PlainTextBody: "hello",
	}
	providerClient := &fakeProviderClient{
		refs:     []domain.MessageRef{{ID: "m-1"}},
		messages: map[string]*domain.MailMessage{"m-1": mail},
	}
	publisher := &fakeInboxEventPublisher{}
	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, &fakeFileStore{}, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	require.NoError(t, cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"}))
	require.Len(t, publisher.published, 1)

	require.NoError(t, cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"}))
	assert.Len(t, publisher.published, 1)
}

func TestSyncAccountCommand_UsesHistoryWhenCursorHasHistoryID(t *testing.T) {
	historyID := "hist-1"
	now := time.Now().UTC()
	cursor, err := domain.NewSyncCursor("acc-1", &now)
	require.NoError(t, err)
	require.NoError(t, cursor.AdvanceHistory(historyID))

	repo := newFakeInboxRepo()
	repo.cursors["acc-1"] = cursor
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	mail := &domain.MailMessage{
		ID:            "m-hist",
		ThreadID:      "t-1",
		Subject:       "from history",
		Sender:        "sender@example.com",
		PlainTextBody: "hello",
	}
	providerClient := &fakeProviderClient{
		historyID: "hist-2",
		historyPage: domain.HistoryPage{
			Changes:      []domain.HistoryChange{{Type: domain.HistoryChangeAdded, MessageID: "m-hist"}},
			NewHistoryID: "hist-2",
		},
		messages: map[string]*domain.MailMessage{"m-hist": mail},
	}
	publisher := &fakeInboxEventPublisher{}
	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, &fakeFileStore{}, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	require.NoError(t, cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"}))
	assert.Equal(t, []string{historyID}, providerClient.listHistoryCalls)
	assert.Empty(t, providerClient.listQueries)
	require.Len(t, publisher.published, 1)
	assert.Equal(t, "hist-2", repo.cursors["acc-1"].HistoryID())
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
	require.NoError(t, err)
	require.Len(t, providerClient.downloadAttachmentCalls, 1)
	require.Len(t, repo.upsertedAttachments, 1)
	assert.Equal(t, "provider-msg-1", providerClient.downloadAttachmentCalls[0].messageID)
	assert.Equal(t, "att-1", providerClient.downloadAttachmentCalls[0].attachmentID)
	assert.Equal(t, "doc.xml", repo.upsertedAttachments[0].Filename)
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
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

	cmd := inboxCommands.NewSyncAccountCommand(repo, repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, publisher, attachmentStore, fakeUnitOfWork{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	err := cmd.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: "acc-1"})
	require.Error(t, err)
	assert.Equal(t, 1, connectionsSvc.markReconnectCalls)

	cursor := repo.cursors["acc-1"]
	require.NotNil(t, cursor)
	assert.Equal(t, domain.SyncCursorStatusError, cursor.Status())
}

func toUnixString(v time.Time) string {
	return strconv.FormatInt(v.Unix(), 10)
}

type fakeInboxRepo struct {
	cursors             map[string]*domain.SyncCursor
	upsertedCursors     []*domain.SyncCursor
	upsertedMessages    []*domain.InboxMessage
	upsertedAttachments []*domain.MessageAttachment
	messagesByKey       map[string]*domain.InboxMessage
	messagesByID        map[string]*domain.InboxMessage
}

func newFakeInboxRepo() *fakeInboxRepo {
	return &fakeInboxRepo{
		cursors:       map[string]*domain.SyncCursor{},
		messagesByKey: map[string]*domain.InboxMessage{},
		messagesByID:  map[string]*domain.InboxMessage{},
	}
}

func (f *fakeInboxRepo) GetSyncCursor(ctx context.Context, connectionID string) (*domain.SyncCursor, error) {
	return f.cursors[connectionID], nil
}

func (f *fakeInboxRepo) UpsertSyncCursor(ctx context.Context, cursor *domain.SyncCursor) error {
	cloned := domain.RehydrateSyncCursor(cursor.Snapshot())
	f.cursors[cursor.ConnectionID()] = cloned
	f.upsertedCursors = append(f.upsertedCursors, cloned)
	return nil
}

func (f *fakeInboxRepo) UpsertInboxMessage(ctx context.Context, msg *domain.InboxMessage) (bool, error) {
	key := msg.ConnectionID() + ":" + msg.ProviderMessageID()
	_, exists := f.messagesByKey[key]
	f.messagesByKey[key] = msg
	f.messagesByID[msg.ID()] = msg
	f.upsertedMessages = append(f.upsertedMessages, msg)
	return !exists, nil
}

func (f *fakeInboxRepo) GetInboxMessageByID(ctx context.Context, messageID string) (*domain.InboxMessage, error) {
	msg, ok := f.messagesByID[messageID]
	if !ok {
		return nil, domain.ErrInboxMessageNotFound
	}
	return msg, nil
}

func (f *fakeInboxRepo) UpdateInboxMessageFlags(ctx context.Context, message *domain.InboxMessage) error {
	f.messagesByID[message.ID()] = message
	return nil
}

func (f *fakeInboxRepo) GetMessageAttachment(ctx context.Context, messageID, attachmentID string) (*domain.MessageAttachment, error) {
	return nil, domain.ErrInboxMessageNotFound
}

func (f *fakeInboxRepo) GetMessageAttachmentByMessageAndSHA(ctx context.Context, messageID, sha256 string) (*domain.MessageAttachment, error) {
	for _, att := range f.upsertedAttachments {
		if att.MessageID == messageID && att.SHA256 == sha256 {
			return att, nil
		}
	}
	return nil, nil
}

func (f *fakeInboxRepo) ListMessageAttachments(ctx context.Context, messageID string) ([]*domain.MessageAttachment, error) {
	return nil, nil
}

func (f *fakeInboxRepo) UpsertMessageAttachment(ctx context.Context, attachment *domain.MessageAttachment) (bool, error) {
	f.upsertedAttachments = append(f.upsertedAttachments, attachment)
	return true, nil
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
	historyID               string
	historyPage             domain.HistoryPage
	historyExpired          bool
	listHistoryCalls        []string
	modifyCalls             []domain.MessageMutation
	modifyErr               error
	trashCalls              []string
	trashErr                error
	sendCalls               []domain.OutgoingMail
	sendErr                 error
	sendID                  string
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

func (f *fakeProviderClient) GetHistoryID(ctx context.Context, userID string) (string, error) {
	return f.historyID, nil
}

func (f *fakeProviderClient) ListHistory(ctx context.Context, userID, startHistoryID string) (domain.HistoryPage, error) {
	f.listHistoryCalls = append(f.listHistoryCalls, startHistoryID)
	if f.historyExpired {
		return domain.HistoryPage{Expired: true}, nil
	}
	return f.historyPage, nil
}

func (f *fakeProviderClient) ModifyMessage(ctx context.Context, userID, messageID string, mutation domain.MessageMutation) error {
	f.modifyCalls = append(f.modifyCalls, mutation)
	return f.modifyErr
}

func (f *fakeProviderClient) TrashMessage(ctx context.Context, userID, messageID string) error {
	f.trashCalls = append(f.trashCalls, messageID)
	return f.trashErr
}

func (f *fakeProviderClient) SendMessage(ctx context.Context, userID string, message domain.OutgoingMail) (string, error) {
	f.sendCalls = append(f.sendCalls, message)
	if f.sendErr != nil {
		return "", f.sendErr
	}
	if f.sendID != "" {
		return f.sendID, nil
	}
	return "sent-1", nil
}

type fakeInboxEventPublisher struct {
	published []platformEvents.BusinessEvent
}

func (f *fakeInboxEventPublisher) Publish(ctx context.Context, event platformEvents.BusinessEvent) error {
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

func (f *fakeFileStore) DownloadFile(ctx context.Context, input platformStorage.DownloadFileInput) error {
	return nil
}

func (f *fakeFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	return false, nil
}

func (f *fakeFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (f *fakeFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, nil
}

func (f *fakeFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, nil
}

type attachmentDownloadCall struct {
	messageID    string
	attachmentID string
}

var _ connectionsApp.InternalService = (*fakeConnectionsInternalService)(nil)
var _ domain.SyncCursorRepository = (*fakeInboxRepo)(nil)
var _ domain.MessageRepository = (*fakeInboxRepo)(nil)
var _ domain.MailProviderClient = (*fakeProviderClient)(nil)
var _ platformEvents.EventBus = (*fakeInboxEventPublisher)(nil)
var _ platformStorage.FileStore = (*fakeFileStore)(nil)
