package events

import (
	"github.com/bowerbird/internal/invoices/adapters/events/handlers"
	commands "github.com/bowerbird/internal/invoices/application/commands"
)

func NewInboxMessageReceivedSubscriber(command *commands.CreateInvoicesFromInboxMessageCommand) *handlers.OnInboxMessageReceived {
	return handlers.NewOnInboxMessageReceived(command)
}
