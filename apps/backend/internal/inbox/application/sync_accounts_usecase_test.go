package application

import (
	"context"
	"errors"
	"testing"

	connectionsapp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

func TestSyncSingleAccountWithResult_ContinuesAfterPayloadRejected(t *testing.T) {
	repo := &fakeInboxRepo{}
	connectionsSvc := &fakeConnectionsInternalService{}
	providerClient := &fakeProviderClient{
		refs: []domain.MessageRef{{ID: "m-1"}, {ID: "m-2"}},
		messages: map[string]*domain.MailMessage{
			"m-1": {
				ID: "m-1",
				Attachments: []domain.MailAttachmentRef{{
					AttachmentID: "a-1",
					Size:         9999999,
				}},
			},
			"m-2": {
				ID:            "m-2",
				ThreadID:      "t-2",
				Subject:       "ok",
				Sender:        "sender@example.com",
				PlainTextBody: "normal",
			},
		},
	}

	uc := NewSyncAccountsUseCase(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, nil, nil)
	uc.maxAttachmentBytes = 1024

	ctx := tenant.WithTenantSlug(context.Background(), "tenant-a")
	account := connectionsapp.ConnectionInfo{ID: "conn-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	result := &SyncAccountsResult{}

	err := uc.SyncSingleAccountWithResult(ctx, "tenant-a", account, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.MessagesSynced != 1 {
		t.Fatalf("expected 1 synced message, got %d", result.MessagesSynced)
	}
	if len(repo.upsertedMessages) != 1 {
		t.Fatalf("expected 1 persisted message, got %d", len(repo.upsertedMessages))
	}
	if connectionsSvc.markReconnectCalls != 0 {
		t.Fatalf("expected no reconnect mark, got %d", connectionsSvc.markReconnectCalls)
	}
}

func TestSyncSingleAccountWithResult_ContinuesAfterInvalidSenderPayload(t *testing.T) {
	repo := &fakeInboxRepo{}
	connectionsSvc := &fakeConnectionsInternalService{}
	providerClient := &fakeProviderClient{
		refs: []domain.MessageRef{{ID: "m-1"}, {ID: "m-2"}},
		messages: map[string]*domain.MailMessage{
			"m-1": {
				ID:            "m-1",
				Sender:        `<img src=x onerror=alert(1)>`,
				PlainTextBody: "bad sender",
			},
			"m-2": {
				ID:            "m-2",
				ThreadID:      "t-2",
				Subject:       "ok",
				Sender:        "Sender <sender@example.com>",
				PlainTextBody: "normal",
			},
		},
	}

	uc := NewSyncAccountsUseCase(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, nil, nil)

	ctx := tenant.WithTenantSlug(context.Background(), "tenant-a")
	account := connectionsapp.ConnectionInfo{ID: "conn-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	result := &SyncAccountsResult{}

	err := uc.SyncSingleAccountWithResult(ctx, "tenant-a", account, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.MessagesSynced != 1 {
		t.Fatalf("expected 1 synced message, got %d", result.MessagesSynced)
	}
	if len(repo.upsertedMessages) != 1 {
		t.Fatalf("expected 1 persisted message, got %d", len(repo.upsertedMessages))
	}
	if repo.upsertedMessages[0].SenderEmail == nil || *repo.upsertedMessages[0].SenderEmail != "sender@example.com" {
		t.Fatalf("expected normalized sender email, got %+v", repo.upsertedMessages[0].SenderEmail)
	}
}

func TestSyncSingleAccountWithResult_ReauthMarksReconnect(t *testing.T) {
	repo := &fakeInboxRepo{}
	connectionsSvc := &fakeConnectionsInternalService{}
	providerClient := &fakeProviderClient{listErr: errors.New("list failed with status 401")}

	uc := NewSyncAccountsUseCase(repo, connectionsSvc, &fakeProviderFactory{client: providerClient}, nil, nil)

	ctx := tenant.WithTenantSlug(context.Background(), "tenant-a")
	account := connectionsapp.ConnectionInfo{ID: "conn-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}
	result := &SyncAccountsResult{}

	err := uc.SyncSingleAccountWithResult(ctx, "tenant-a", account, result)
	if err == nil {
		t.Fatalf("expected sync error")
	}

	if connectionsSvc.markReconnectCalls != 1 {
		t.Fatalf("expected reconnect mark once, got %d", connectionsSvc.markReconnectCalls)
	}
	if result.Failures != 1 {
		t.Fatalf("expected one failure, got %d", result.Failures)
	}
}

type fakeInboxRepo struct {
	cursor           *domain.InboxSyncCursor
	upsertedMessages []*domain.EmailMessage
}

func (f *fakeInboxRepo) GetSyncCursor(ctx context.Context, connectionID string) (*domain.InboxSyncCursor, error) {
	return f.cursor, nil
}

func (f *fakeInboxRepo) UpsertSyncCursor(ctx context.Context, cursor *domain.InboxSyncCursor) error {
	f.cursor = cursor
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
	markReconnectCalls int
}

func (f *fakeConnectionsInternalService) GetActiveConnections(ctx context.Context) ([]connectionsapp.ConnectionInfo, error) {
	return nil, nil
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
	refs     []domain.MessageRef
	messages map[string]*domain.MailMessage
	listErr  error
}

func (f *fakeProviderClient) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	if f.listErr != nil {
		return nil, "", f.listErr
	}
	return f.refs, "", nil
}

func (f *fakeProviderClient) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	message, ok := f.messages[messageID]
	if !ok {
		return nil, errors.New("message not found")
	}
	return message, nil
}

func (f *fakeProviderClient) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
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

var _ connectionsapp.InternalService = (*fakeConnectionsInternalService)(nil)
var _ domain.Repository = (*fakeInboxRepo)(nil)
var _ domain.MailProviderClient = (*fakeProviderClient)(nil)
