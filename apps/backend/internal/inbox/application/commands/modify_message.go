package commands

import (
	"context"
	"fmt"
	"time"

	connectionsApp "github.com/bowerbird/internal/connections/application"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/bowerbird/internal/platform/tenant"
)

type MessageAction string

const (
	MessageActionRead    MessageAction = "read"
	MessageActionUnread  MessageAction = "unread"
	MessageActionStar    MessageAction = "star"
	MessageActionUnstar  MessageAction = "unstar"
	MessageActionArchive MessageAction = "archive"
	MessageActionTrash   MessageAction = "trash"
)

type ModifyMessageCommand struct {
	messageRepo        domain.MessageRepository
	connectionsService connectionsApp.InternalService
	providerFactory    ProviderClientFactory
}

func NewModifyMessageCommand(
	messageRepo domain.MessageRepository,
	connectionsService connectionsApp.InternalService,
	providerFactory ProviderClientFactory,
) *ModifyMessageCommand {
	return &ModifyMessageCommand{
		messageRepo:        messageRepo,
		connectionsService: connectionsService,
		providerFactory:    providerFactory,
	}
}

func (c *ModifyMessageCommand) Execute(ctx context.Context, messageID string, action MessageAction) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return err
	}

	message, err := c.messageRepo.GetInboxMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	account, credentialsJSON, err := decryptActiveAccount(ctx, c.connectionsService, message.ConnectionID)
	if err != nil {
		return err
	}

	client, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	now := time.Now().UTC()
	switch action {
	case MessageActionRead:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID, domain.MessageMutation{RemoveLabelIDs: []string{"UNREAD"}}); err != nil {
			return fmt.Errorf("mark read: %w", err)
		}
		message.IsRead = true
	case MessageActionUnread:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID, domain.MessageMutation{AddLabelIDs: []string{"UNREAD"}}); err != nil {
			return fmt.Errorf("mark unread: %w", err)
		}
		message.IsRead = false
	case MessageActionStar:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID, domain.MessageMutation{AddLabelIDs: []string{"STARRED"}}); err != nil {
			return fmt.Errorf("star message: %w", err)
		}
		message.IsStarred = true
	case MessageActionUnstar:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID, domain.MessageMutation{RemoveLabelIDs: []string{"STARRED"}}); err != nil {
			return fmt.Errorf("unstar message: %w", err)
		}
		message.IsStarred = false
	case MessageActionArchive:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID, domain.MessageMutation{RemoveLabelIDs: []string{"INBOX"}}); err != nil {
			return fmt.Errorf("archive message: %w", err)
		}
		if message.Folder == domain.MailFolderInbox {
			message.Folder = domain.MailFolderArchive
		}
	case MessageActionTrash:
		if err := client.TrashMessage(ctx, "me", message.ProviderMessageID); err != nil {
			return fmt.Errorf("trash message: %w", err)
		}
		message.Folder = domain.MailFolderTrash
	default:
		return fmt.Errorf("unsupported message action %q", action)
	}

	message.UpdatedAt = now
	return c.messageRepo.UpdateInboxMessageFlags(ctx, message)
}
