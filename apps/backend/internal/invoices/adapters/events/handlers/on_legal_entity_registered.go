package handlers

import (
	"context"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	commands "github.com/bowerbird/internal/invoices/application/commands"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/tenant"
)

type OnLegalEntityRegistered struct {
	command *commands.BackfillInboxInvoicesCommand
}

func NewOnLegalEntityRegistered(command *commands.BackfillInboxInvoicesCommand) *OnLegalEntityRegistered {
	return &OnLegalEntityRegistered{command: command}
}

func (h *OnLegalEntityRegistered) DetailType() string {
	return contractEvents.LegalEntityRegisteredDetailType
}

func (h *OnLegalEntityRegistered) Handle(ctx context.Context, event platformEvents.IntegrationEvent) error {
	if h.command == nil {
		return nil
	}
	decoded, err := contractEvents.UnmarshalLegalEntityRegistered(event.Detail)
	if err != nil {
		return err
	}
	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	return h.command.Enqueue(msgCtx)
}
