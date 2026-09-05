package commands

import (
	"context"
	"fmt"
	"time"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/tenant"
)

type SendMessageInput struct {
	AccountID string
	To        []string
	Cc        []string
	Bcc       []string
	Subject   string
	BodyText  string
	BodyHTML  string
	ThreadID  string
	InReplyTo string
}

type SendMessageCommand struct {
	messageRepo        domain.MessageRepository
	connectionsService connectionsapi.InternalService
	providerFactory    ProviderClientFactory
	idGenerator        func() string
}

func NewSendMessageCommand(
	messageRepo domain.MessageRepository,
	connectionsService connectionsapi.InternalService,
	providerFactory ProviderClientFactory,
) *SendMessageCommand {
	if messageRepo == nil {
		panic("message repository is required")
	}
	if connectionsService == nil {
		panic("connections service is required")
	}
	if providerFactory == nil {
		panic("provider factory is required")
	}
	return &SendMessageCommand{
		messageRepo:        messageRepo,
		connectionsService: connectionsService,
		providerFactory:    providerFactory,
		idGenerator:        id.NewULID,
	}
}

func (c *SendMessageCommand) Execute(ctx context.Context, input SendMessageInput) (*domain.InboxMessage, error) {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return nil, err
	}
	if len(input.To) == 0 {
		return nil, domain.ErrOutgoingMailToRequired
	}
	if input.AccountID == "" {
		return nil, fmt.Errorf("account id is required")
	}

	account, credentialsJSON, err := decryptActiveAccount(ctx, c.connectionsService, input.AccountID)
	if err != nil {
		return nil, err
	}

	client, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("build provider client: %w", err)
	}

	outgoing := domain.OutgoingMail{
		To:        input.To,
		Cc:        input.Cc,
		Bcc:       input.Bcc,
		Subject:   input.Subject,
		BodyText:  input.BodyText,
		BodyHTML:  input.BodyHTML,
		ThreadID:  input.ThreadID,
		InReplyTo: input.InReplyTo,
	}

	providerID, err := client.SendMessage(ctx, "me", outgoing)
	if err != nil {
		return nil, fmt.Errorf("send provider message: %w", err)
	}

	now := time.Now().UTC()
	threadID := input.ThreadID
	message, err := domain.NewInboxMessageAsSynced(domain.NewInboxMessageInput{
		ID:                c.idGenerator(),
		ConnectionID:      account.ID,
		ProviderMessageID: providerID,
		ProviderThreadID:  optionalStringPointer(threadID),
		Subject:           optionalStringPointer(input.Subject),
		SenderEmail:       optionalStringPointer(account.ProviderAccountEmail),
		ToEmails:          input.To,
		CcEmails:          input.Cc,
		BccEmails:         input.Bcc,
		Folder:            domain.MailFolderSent,
		IsRead:            true,
		ReceivedAt:        &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return nil, err
	}

	if _, err := c.messageRepo.UpsertInboxMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("save sent message: %w", err)
	}

	return message, nil
}

func optionalStringPointer(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}
