package handlers

import (
	"context"

	contractEvents "github.com/atta/internal/contracts/events"
	commands "github.com/atta/internal/invoices/application/commands"
	platformEvents "github.com/atta/internal/platform/events"
	"github.com/atta/internal/platform/tenant"
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
