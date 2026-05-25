package router

import (
	"context"
	"log"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
)

type LogRouter struct{}

func NewLogRouter() *LogRouter {
	return &LogRouter{}
}

func (r *LogRouter) RouteInboxInvoiceCandidate(ctx context.Context, event contractevents.InboxMessageReceived) error {
	log.Printf(
		"invoicing candidate routed: tenant=%s account=%s provider=%s provider_message_id=%s attachments=%d",
		event.TenantSlug,
		event.AccountID,
		event.Provider,
		event.ProviderMessageID,
		len(event.AttachmentRefs),
	)
	return nil
}
