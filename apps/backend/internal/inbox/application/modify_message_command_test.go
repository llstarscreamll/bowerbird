package application_test

import (
	"context"
	"testing"
	"time"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModifyMessageCommand_ArchivesViaProvider(t *testing.T) {
	repo := newFakeInboxRepo()
	now := time.Now().UTC()
	message, err := domain.NewInboxMessageAsSynced(domain.NewInboxMessageInput{
		ID:                "msg-1",
		ConnectionID:      "acc-1",
		ProviderMessageID: "prov-1",
		Folder:            domain.MailFolderInbox,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	require.NoError(t, err)
	repo.messagesByID[message.ID()] = message
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "user@gmail.com"}},
	}
	providerClient := &fakeProviderClient{}
	cmd := inboxCommands.NewModifyMessageCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	require.NoError(t, cmd.Execute(ctx, "msg-1", inboxCommands.MessageActionArchive))
	require.Len(t, providerClient.modifyCalls, 1)
	assert.Equal(t, []string{"INBOX"}, providerClient.modifyCalls[0].RemoveLabelIDs)
	assert.Equal(t, domain.MailFolderArchive, repo.messagesByID["msg-1"].Folder())
}

func TestSendMessageCommand_RequiresRecipient(t *testing.T) {
	cmd := inboxCommands.NewSendMessageCommand(newFakeInboxRepo(), &fakeConnectionsInternalService{}, &fakeProviderFactory{client: &fakeProviderClient{}})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	_, err := cmd.Execute(ctx, inboxCommands.SendMessageInput{AccountID: "acc-1", Subject: "Hi"})
	require.ErrorIs(t, err, domain.ErrOutgoingMailToRequired)
}

func TestSendMessageCommand_PersistsSentCopy(t *testing.T) {
	repo := newFakeInboxRepo()
	connectionsSvc := &fakeConnectionsInternalService{
		activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail", ProviderAccountEmail: "me@gmail.com"}},
	}
	providerClient := &fakeProviderClient{sendID: "gmail-sent-9"}
	cmd := inboxCommands.NewSendMessageCommand(repo, connectionsSvc, &fakeProviderFactory{client: providerClient})
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	message, err := cmd.Execute(ctx, inboxCommands.SendMessageInput{
		AccountID: "acc-1",
		To:        []string{"to@example.com"},
		Subject:   "Hello",
		BodyText:  "body",
	})
	require.NoError(t, err)
	require.NotNil(t, message)
	assert.Equal(t, "gmail-sent-9", message.ProviderMessageID())
	assert.Equal(t, domain.MailFolderSent, message.Folder())
	require.Len(t, providerClient.sendCalls, 1)
	assert.Equal(t, []string{"to@example.com"}, providerClient.sendCalls[0].To)
}
