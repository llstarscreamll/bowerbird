package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/bowerbird/internal/inbox/domain"
)

type fakeClient struct{}

func (c fakeClient) ListMessages(ctx context.Context, opts domain.ListMessagesOptions) ([]domain.MessageRef, string, error) {
	return nil, "", nil
}

func (c fakeClient) GetMessage(ctx context.Context, userID, messageID string) (*domain.MailMessage, error) {
	return nil, nil
}

func (c fakeClient) DownloadAttachment(ctx context.Context, userID, messageID, attachmentID string) ([]byte, error) {
	return nil, nil
}

func (c fakeClient) DownloadMessageAttachments(ctx context.Context, userID, messageID string, refs []domain.MailAttachmentRef) ([]domain.DownloadedMailAttachment, error) {
	return nil, nil
}

func (c fakeClient) CreateLabel(ctx context.Context, userID, labelName string) (string, error) {
	return "", nil
}

func (c fakeClient) AddLabelToMessage(ctx context.Context, userID, messageID, labelID string) error {
	return nil
}

func TestFactoryBuildUsesRegisteredProvider(t *testing.T) {
	f := NewFactory()
	f.Register(domain.ProviderGmail, func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error) {
		return fakeClient{}, nil
	})

	client, err := f.Build(context.Background(), "GMAIL", []byte(`{"refresh_token":"x"}`))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryBuildFailsForUnsupportedProvider(t *testing.T) {
	f := NewFactory()
	if _, err := f.Build(context.Background(), "yahoo", []byte(`{}`)); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestFactoryBuildPropagatesBuilderError(t *testing.T) {
	f := NewFactory()
	want := errors.New("boom")
	f.Register("gmail", func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error) {
		return nil, want
	})

	_, err := f.Build(context.Background(), "gmail", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
