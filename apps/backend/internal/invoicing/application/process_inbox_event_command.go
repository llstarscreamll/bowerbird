package application

import (
	"context"
	"log/slog"
	"strings"

	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
)

type InboxInvoiceRouter interface {
	RouteInboxInvoiceCandidate(ctx context.Context, event contractEvents.InboxMessageReceived) error
}

type ProcessInboxEventCommand struct {
	router InboxInvoiceRouter
	logger *slog.Logger
}

func NewProcessInboxEventCommand(router InboxInvoiceRouter) *ProcessInboxEventCommand {
	return &ProcessInboxEventCommand{router: router, logger: slog.Default()}
}

func (cmd *ProcessInboxEventCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) error {
	if !isInvoiceCandidate(event) {
		cmd.logger.Info("invoicing event skipped: not invoice candidate", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	if cmd.router == nil {
		cmd.logger.Info("invoicing router not configured", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	if err := cmd.router.RouteInboxInvoiceCandidate(ctx, event); err != nil {
		return err
	}
	cmd.logger.Info(
		"invoicing event routed",
		"tenant_slug",
		event.TenantSlug,
		"message_id",
		event.MessageInternalID,
		"provider_message_id",
		event.ProviderMessageID,
	)
	return nil
}

func isInvoiceCandidate(event contractEvents.InboxMessageReceived) bool {
	for _, ref := range event.AttachmentRefs {
		filename := strings.ToLower(ref.Filename)
		if strings.HasSuffix(filename, ".xml") || strings.HasSuffix(filename, ".pdf") {
			return true
		}
	}

	subject := strings.ToLower(event.Subject)
	if strings.Contains(subject, "factura") || strings.Contains(subject, "invoice") {
		return true
	}

	return false
}
