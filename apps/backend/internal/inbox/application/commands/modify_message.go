package commands

import (
	"context"
	"errors"
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

var ErrInvalidMessageAction = errors.New("unsupported message action")

func IsValidMessageAction(action MessageAction) bool {
	switch action {
	case MessageActionRead, MessageActionUnread, MessageActionStar, MessageActionUnstar, MessageActionArchive, MessageActionTrash:
		return true
	default:
		return false
	}
}

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

	if !IsValidMessageAction(action) {
		return fmt.Errorf("%w: %s", ErrInvalidMessageAction, action)
	}

	message, err := c.messageRepo.GetInboxMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	account, credentialsJSON, err := decryptAccount(ctx, c.connectionsService, message.ConnectionID())
	if err != nil {
		return err
	}

	client, err := c.providerFactory.Build(ctx, account.Provider, credentialsJSON)
	if err != nil {
		return fmt.Errorf("build provider client: %w", err)
	}

	now := time.Now().UTC()
	providerErr := func(op string, err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("%s: %w", op, classifySyncError(account, err))
	}
	switch action {
	case MessageActionRead:
		if err := providerErr("mark read", client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"UNREAD"}})); err != nil {
			return err
		}
		message.MarkAsRead(now)
	case MessageActionUnread:
		if err := providerErr("mark unread", client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{AddLabelIDs: []string{"UNREAD"}})); err != nil {
			return err
		}
		message.MarkAsUnread(now)
	case MessageActionStar:
		if err := providerErr("star message", client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{AddLabelIDs: []string{"STARRED"}})); err != nil {
			return err
		}
		message.Star(now)
	case MessageActionUnstar:
		if err := providerErr("unstar message", client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"STARRED"}})); err != nil {
			return err
		}
		message.Unstar(now)
	case MessageActionArchive:
		if err := providerErr("archive message", client.ModifyMessage(ctx, "me", message.ProviderMessageID(), domain.MessageMutation{RemoveLabelIDs: []string{"INBOX"}})); err != nil {
			return err
		}
		message.Archive(now)
	case MessageActionTrash:
		if err := providerErr("trash message", client.TrashMessage(ctx, "me", message.ProviderMessageID())); err != nil {
			return err
		}
		message.MoveToTrash(now)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidMessageAction, action)
	}

	return c.messageRepo.UpdateInboxMessageFlags(ctx, message)
}
