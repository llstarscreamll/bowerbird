package commands

import (
	"context"
	"fmt"
	"time"

	connectionsapi "github.com/bowerbird/internal/connections/api"
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
	connectionsService connectionsapi.InternalService
	providerFactory    ProviderClientFactory
}

func NewModifyMessageCommand(
	messageRepo domain.MessageRepository,
	connectionsService connectionsapi.InternalService,
	providerFactory ProviderClientFactory,
) *ModifyMessageCommand {
	if messageRepo == nil {
		panic("message repository is required")
	}
	if connectionsService == nil {
		panic("connections service is required")
	}
	if providerFactory == nil {
		panic("provider factory is required")
	}
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

	account, credentialsJSON, err := decryptActiveAccount(ctx, c.connectionsService, message.ConnectionID())
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
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"UNREAD"}}); err != nil {
			return fmt.Errorf("mark read: %w", err)
		}
		message.MarkAsRead(now)
	case MessageActionUnread:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{AddLabelIDs: []string{"UNREAD"}}); err != nil {
			return fmt.Errorf("mark unread: %w", err)
		}
		message.MarkAsUnread(now)
	case MessageActionStar:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{AddLabelIDs: []string{"STARRED"}}); err != nil {
			return fmt.Errorf("star message: %w", err)
		}
		message.Star(now)
	case MessageActionUnstar:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"STARRED"}}); err != nil {
			return fmt.Errorf("unstar message: %w", err)
		}
		message.Unstar(now)
	case MessageActionArchive:
		if err := client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"INBOX"}}); err != nil {
			return fmt.Errorf("archive message: %w", err)
		}
		message.Archive(now)
	case MessageActionTrash:
		if err := client.TrashMessage(ctx, "me", message.ProviderMessageID()); err != nil {
			return fmt.Errorf("trash message: %w", err)
		}
		message.MoveToTrash(now)
	default:
		return fmt.Errorf("unsupported message action %q", action)
	}

	return c.messageRepo.UpdateInboxMessageFlags(ctx, message)
}
