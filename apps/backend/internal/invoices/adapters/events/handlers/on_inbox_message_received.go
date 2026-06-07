package handlers

import (
	"context"

	awsEvents "github.com/aws/aws-lambda-go/events"
	contractEvents "github.com/bowerbird/internal/contracts/events"
	commands "github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/platform/tenant"
)

type OnInboxMessageReceived struct {
	command *commands.CreateInvoicesFromInboxMessageCommand
}

func NewOnInboxMessageReceived(command *commands.CreateInvoicesFromInboxMessageCommand) *OnInboxMessageReceived {
	return &OnInboxMessageReceived{command: command}
}

func (h *OnInboxMessageReceived) DetailType() string {
	return contractEvents.InboxMessageReceivedDetailType
}

func (h *OnInboxMessageReceived) HandleEventBridge(ctx context.Context, event awsEvents.CloudWatchEvent) error {
	if h.command == nil {
		return nil
	}

	decoded, err := contractEvents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantID)
	return h.command.Execute(msgCtx, decoded)
}
