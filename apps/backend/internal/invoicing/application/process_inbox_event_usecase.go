package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/observability"
)

type InboxInvoiceRouter interface {
	RouteInboxInvoiceCandidate(ctx context.Context, event contractevents.InboxMessageReceived) error
}

type ProcessInboxEventUseCase struct {
	router  InboxInvoiceRouter
	logger  *slog.Logger
	metrics observability.Metrics
}

func NewProcessInboxEventUseCase(router InboxInvoiceRouter) *ProcessInboxEventUseCase {
	return &ProcessInboxEventUseCase{router: router, logger: slog.Default(), metrics: observability.NoopMetrics{}}
}

func (u *ProcessInboxEventUseCase) Process(ctx context.Context, event contractevents.InboxMessageReceived) error {
	startedAt := time.Now()
	if !isInvoiceCandidate(event) {
		u.logger.Info("invoicing event skipped: not invoice candidate", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}
	u.metrics.IncCounter("invoicing_documents_classified_total", map[string]string{"tenant_slug": event.TenantSlug, "result": "candidate"})

	if u.router == nil {
		u.logger.Info("invoicing router not configured", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	if err := u.router.RouteInboxInvoiceCandidate(ctx, event); err != nil {
		u.metrics.IncCounter("invoicing_processing_errors_total", map[string]string{"tenant_slug": event.TenantSlug})
		return err
	}
	u.metrics.ObserveDuration("invoicing_processing_latency_ms", time.Since(startedAt), map[string]string{"tenant_slug": event.TenantSlug})
	u.logger.Info(
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

func isInvoiceCandidate(event contractevents.InboxMessageReceived) bool {
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
