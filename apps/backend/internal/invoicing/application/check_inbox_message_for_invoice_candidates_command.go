package application

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
)

type CheckInboxMessageForInvoiceCandidatesCommand struct {
	publisher events.BusinessEventPublisher
	logger    *slog.Logger
	now       func() time.Time
	newID     func() string
}

func NewCheckInboxMessageForInvoiceCandidatesCommand(publisher events.BusinessEventPublisher) *CheckInboxMessageForInvoiceCandidatesCommand {
	return &CheckInboxMessageForInvoiceCandidatesCommand{
		publisher: publisher,
		logger:    slog.Default(),
		now:       time.Now,
		newID:     id.NewULID,
	}
}

func (cmd *CheckInboxMessageForInvoiceCandidatesCommand) Execute(ctx context.Context, event contractEvents.InboxMessageReceived) error {
	if !hasInvoiceKeyword(event.Subject, event.Body) {
		cmd.logger.Info("invoicing event skipped: missing invoice keyword", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	if !hasSupportedAttachment(event.AttachmentRefs) {
		cmd.logger.Info("invoicing event skipped: missing supported attachments", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	if cmd.publisher == nil {
		cmd.logger.Info("invoicing candidate detected but publisher not configured", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID)
		return nil
	}

	job := contractEvents.InvoiceExtractionRequested{
		EventID:           cmd.newID(),
		OccurredAt:        cmd.now().UTC().Format(time.RFC3339Nano),
		TenantSlug:        event.TenantSlug,
		SourceMessageID:   event.MessageInternalID,
		ProviderMessageID: event.ProviderMessageID,
		AttachmentRefs:    append([]contractEvents.AttachmentRef(nil), event.AttachmentRefs...),
	}

	payload, err := contractEvents.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return err
	}

	err = cmd.publisher.PublishBusinessEvent(ctx, events.BusinessEvent{
		Source:     contractEvents.InvoiceExtractionRequestedSource,
		DetailType: contractEvents.InvoiceExtractionRequestedDetailType,
		Detail:     payload,
	})
	if err != nil {
		return err
	}

	cmd.logger.Info("invoice extraction job queued", "tenant_slug", event.TenantSlug, "message_id", event.MessageInternalID, "attachments", len(event.AttachmentRefs))
	return nil
}

func hasSupportedAttachment(refs []contractEvents.AttachmentRef) bool {
	for _, ref := range refs {
		ext := strings.ToLower(filepath.Ext(ref.Filename))
		if ext == ".xml" || ext == ".pdf" || ext == ".zip" {
			return true
		}

		mime := strings.ToLower(strings.TrimSpace(ref.MimeType))
		if strings.Contains(mime, "pdf") || strings.Contains(mime, "xml") || strings.Contains(mime, "zip") {
			return true
		}
	}

	return false
}

func hasInvoiceKeyword(subject, body string) bool {
	combined := strings.ToLower(strings.TrimSpace(subject + "\n" + body))
	if combined == "" {
		return false
	}

	keywords := []string{
		"factura electronica",
		"facturación electrónica",
		"factura electrónica",
		"facturacion electronica",
		"facturacion",
		"factura",
		"invoice",
	}

	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}
