package application

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	platformcrypto "github.com/money-path/bowerbird/apps/backend/internal/platform/crypto"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type fakeInboxRepo struct {
	accounts                  []*domain.ConnectedAccount
	upsertedMessages          []*domain.EmailMessage
	updatedStates             []updatedAccountState
	upsertByProviderMessageID map[string]bool
}

type updatedAccountState struct {
	accountID string
	status    string
}

func (r *fakeInboxRepo) CreateConnectedAccount(ctx context.Context, account *domain.ConnectedAccount) error {
	return nil
}

func (r *fakeInboxRepo) GetConnectedAccountByID(ctx context.Context, accountID string) (*domain.ConnectedAccount, error) {
	return nil, domain.ErrConnectedAccountNotFound
}

func (r *fakeInboxRepo) ListConnectedAccountsByStatus(ctx context.Context, status string) ([]*domain.ConnectedAccount, error) {
	return r.accounts, nil
}

func (r *fakeInboxRepo) UpdateConnectedAccountSyncState(ctx context.Context, accountID, status string, lastSyncedAt *time.Time, lastError *string, updatedAt time.Time) error {
	r.updatedStates = append(r.updatedStates, updatedAccountState{accountID: accountID, status: status})
	return nil
}

func (r *fakeInboxRepo) UpsertEmailMessage(ctx context.Context, message *domain.EmailMessage) (bool, error) {
	r.upsertedMessages = append(r.upsertedMessages, message)
	if r.upsertByProviderMessageID == nil {
		return true, nil
	}
	inserted, ok := r.upsertByProviderMessageID[message.ProviderMessageID]
	if !ok {
		return true, nil
	}
	return inserted, nil
}

func (r *fakeInboxRepo) GetEmailMessageByProviderID(ctx context.Context, accountID, providerMessageID string) (*domain.EmailMessage, error) {
	return nil, domain.ErrEmailMessageNotFound
}

func (r *fakeInboxRepo) UpsertEmailAttachment(ctx context.Context, attachment *domain.EmailAttachment) (bool, error) {
	return true, nil
}

func (r *fakeInboxRepo) ListEmailAttachmentsByMessageID(ctx context.Context, messageID string) ([]*domain.EmailAttachment, error) {
	return nil, nil
}

type fakeProviderFactory struct {
	clients map[string]domain.MailProviderClient
	err     error
}

func (f *fakeProviderFactory) Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	client, ok := f.clients[provider]
	if !ok {
		return nil, errors.New("unsupported provider")
	}
	return client, nil
}

type fakeMailProviderClient struct {
	pages [][]domain.MessageRef
	msgs  map[string]*domain.MailMessage
	err   error
	idx   int
}

func (c *fakeMailProviderClient) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	if c.err != nil {
		return nil, "", c.err
	}
	if c.idx >= len(c.pages) {
		return nil, "", nil
	}

	refs := c.pages[c.idx]
	c.idx++
	next := ""
	if c.idx < len(c.pages) {
		next = "next"
	}
	return refs, next, nil
}

func (c *fakeMailProviderClient) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	msg, ok := c.msgs[messageID]
	if !ok {
		return nil, errors.New("message not found")
	}
	return msg, nil
}

func (c *fakeMailProviderClient) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
	return nil, nil
}

func (c *fakeMailProviderClient) DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []domain.MailAttachmentRef) ([]domain.DownloadedMailAttachment, error) {
	return nil, nil
}

type fakePublisher struct {
	events []domain.InboxMessageReceived
	err    error
}

type fakeAttachmentStorage struct{}

func (s *fakeAttachmentStorage) StoreAttachment(ctx context.Context, input StoreAttachmentInput) (*StoredAttachment, error) {
	if input.ConnectedAccountID == "" {
		return nil, errors.New("missing connected account id")
	}
	if input.AttachmentID == "" {
		return nil, errors.New("missing attachment id")
	}
	return &StoredAttachment{S3Key: "tenant/t/inbox/account/messages/msg/attachments/att.bin", SHA256: "hash", SizeBytes: int64(len(input.Data)), Uploaded: true}, nil
}

func (p *fakePublisher) PublishInboxMessageReceived(ctx context.Context, event domain.InboxMessageReceived) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}

func TestSyncAccountsUseCaseRunProcessesMultipleAccountsAndPublishesEvents(t *testing.T) {
	credentialsService := newTestCredentialsService(t)
	encA, _ := credentialsService.EncryptForStorage([]byte(`{"refresh_token":"a"}`))
	encB, _ := credentialsService.EncryptForStorage([]byte(`{"refresh_token":"b"}`))

	repo := &fakeInboxRepo{
		accounts: []*domain.ConnectedAccount{
			{ID: "acc1", Provider: domain.ProviderGmail, Status: "active", EncryptedCredentials: encA},
			{ID: "acc2", Provider: domain.ProviderGmail, Status: "active", EncryptedCredentials: encB},
		},
		upsertByProviderMessageID: map[string]bool{"m1": true, "m2": false, "m3": true},
	}

	client := &fakeMailProviderClient{
		pages: [][]domain.MessageRef{
			{{ID: "m1", ThreadID: "t1"}, {ID: "m2", ThreadID: "t2"}},
			{{ID: "m3", ThreadID: "t3"}},
		},
		msgs: map[string]*domain.MailMessage{
			"m1": {ID: "m1", ThreadID: "t1", Subject: "Factura A", Sender: "a@vendor.com"},
			"m2": {ID: "m2", ThreadID: "t2", Subject: "Factura B", Sender: "b@vendor.com"},
			"m3": {ID: "m3", ThreadID: "t3", Subject: "Factura C", Sender: "c@vendor.com"},
		},
	}

	factory := &fakeProviderFactory{clients: map[string]domain.MailProviderClient{domain.ProviderGmail: client}}
	publisher := &fakePublisher{}

	uc := NewSyncAccountsUseCase(repo, credentialsService, factory, publisher, &fakeAttachmentStorage{})
	uc.now = func() time.Time { return time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC) }
	seq := 0
	uc.newID = func() string {
		seq++
		return "id-" + string(rune('a'+seq))
	}

	ctx := tenant.WithTenantSlug(context.Background(), "tenant_1")
	result, err := uc.Run(ctx)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.AccountsProcessed != 2 {
		t.Fatalf("expected 2 accounts processed, got %d", result.AccountsProcessed)
	}
	if result.MessagesSynced != 2 {
		t.Fatalf("expected 2 messages synced, got %d", result.MessagesSynced)
	}
	if result.EventsPublished != 2 {
		t.Fatalf("expected 2 published events, got %d", result.EventsPublished)
	}

	if len(publisher.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(publisher.events))
	}
}

func TestSyncAccountsUseCaseRunMarksAccountErrorAndContinues(t *testing.T) {
	credentialsService := newTestCredentialsService(t)
	encA, _ := credentialsService.EncryptForStorage([]byte(`{"refresh_token":"a"}`))
	encB := []byte("invalid-ciphertext")

	repo := &fakeInboxRepo{
		accounts: []*domain.ConnectedAccount{
			{ID: "acc1", Provider: domain.ProviderGmail, Status: "active", EncryptedCredentials: encA},
			{ID: "acc2", Provider: domain.ProviderGmail, Status: "active", EncryptedCredentials: encB},
		},
	}

	goodClient := &fakeMailProviderClient{
		pages: [][]domain.MessageRef{{{ID: "m1", ThreadID: "t1"}}},
		msgs:  map[string]*domain.MailMessage{"m1": {ID: "m1", ThreadID: "t1", Subject: "Factura"}},
	}

	factory := &fakeProviderFactory{clients: map[string]domain.MailProviderClient{domain.ProviderGmail: goodClient}}
	publisher := &fakePublisher{}

	uc := NewSyncAccountsUseCase(repo, credentialsService, factory, publisher, &fakeAttachmentStorage{})
	uc.now = func() time.Time { return time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC) }

	ctx := tenant.WithTenantSlug(context.Background(), "tenant_1")
	result, err := uc.Run(ctx)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Failures != 1 {
		t.Fatalf("expected 1 failure, got %d", result.Failures)
	}

	if len(repo.updatedStates) != 2 {
		t.Fatalf("expected 2 sync state updates, got %d", len(repo.updatedStates))
	}
}

func newTestCredentialsService(t *testing.T) *CredentialsService {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	cipher, err := platformcrypto.NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}
	return NewCredentialsService(cipher)
}
