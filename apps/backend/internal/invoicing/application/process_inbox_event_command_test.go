package application

import (
	"context"
	"testing"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
)

type fakeRouter struct {
	routed []contractevents.InboxMessageReceived
}

func (r *fakeRouter) RouteInboxInvoiceCandidate(ctx context.Context, event contractevents.InboxMessageReceived) error {
	r.routed = append(r.routed, event)
	return nil
}

func TestProcessRoutesInvoiceCandidateEvent(t *testing.T) {
	router := &fakeRouter{}
	uc := NewProcessInboxEventCommand(router)

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "factura.pdf"},
		},
	})
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}

	if len(router.routed) != 1 {
		t.Fatalf("expected 1 routed event, got %d", len(router.routed))
	}
}

func TestProcessSkipsNonInvoiceCandidateEvent(t *testing.T) {
	router := &fakeRouter{}
	uc := NewProcessInboxEventCommand(router)

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		Subject:           "meeting notes",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "notes.txt"},
		},
	})
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}

	if len(router.routed) != 0 {
		t.Fatalf("expected 0 routed events, got %d", len(router.routed))
	}
}
