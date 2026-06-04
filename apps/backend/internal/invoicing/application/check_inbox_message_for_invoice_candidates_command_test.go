package application

import (
	"context"
	"encoding/json"
	"testing"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	platformevents "github.com/money-path/bowerbird/apps/backend/internal/platform/events"
)

type fakeBusinessPublisher struct {
	events []platformevents.BusinessEvent
}

func (p *fakeBusinessPublisher) PublishBusinessEvent(ctx context.Context, event platformevents.BusinessEvent) error {
	p.events = append(p.events, event)
	return nil
}

func TestCheckQueuesInvoiceExtractionJob(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCheckInboxMessageForInvoiceCandidatesCommand(publisher)
	uc.newID = func() string { return "evt_1" }

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_src_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "provider_msg_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica de mayo",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "factura.pdf"},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 queued event, got %d", len(publisher.events))
	}
	if publisher.events[0].DetailType != contractevents.InvoiceExtractionRequestedDetailType {
		t.Fatalf("expected detail type %q, got %q", contractevents.InvoiceExtractionRequestedDetailType, publisher.events[0].DetailType)
	}

	var queued contractevents.InvoiceExtractionRequested
	if err := json.Unmarshal(publisher.events[0].Detail, &queued); err != nil {
		t.Fatalf("decode queued event: %v", err)
	}
	if queued.SourceMessageID != "m_1" {
		t.Fatalf("expected source message id m_1, got %q", queued.SourceMessageID)
	}
}

func TestCheckSkipsNonCandidates(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCheckInboxMessageForInvoiceCandidatesCommand(publisher)

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		Subject:           "meeting notes from vendor",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "notes.txt"},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(publisher.events) != 0 {
		t.Fatalf("expected 0 queued events, got %d", len(publisher.events))
	}
}
