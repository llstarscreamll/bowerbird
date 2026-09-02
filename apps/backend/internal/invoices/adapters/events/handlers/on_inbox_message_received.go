package handlers

import (
	"context"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	commands "github.com/bowerbird/internal/invoices/application/commands"
	platformEvents "github.com/bowerbird/internal/platform/events"
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

func (h *OnInboxMessageReceived) Handle(ctx context.Context, event platformEvents.IntegrationEvent) error {
	if h.command == nil {
		return nil
	}

	decoded, err := contractEvents.UnmarshalInboxMessageReceived(event.Detail)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantID)
	return h.command.Execute(msgCtx, decoded)
}
