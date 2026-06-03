package events

import (
	"context"
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingapp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
)

type fakeRouter struct {
	routed int
}

func (r *fakeRouter) RouteInboxInvoiceCandidate(ctx context.Context, event contractevents.InboxMessageReceived) error {
	r.routed++
	return nil
}

func TestSubscriberRoutesInboxMessageReceivedEvent(t *testing.T) {
	router := &fakeRouter{}
	uc := invoicingapp.NewProcessInboxEventCommand(router)
	subscriber := NewInboxMessageReceivedSubscriber(uc)

	detail, err := contractevents.MarshalInboxMessageReceived(contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "factura.xml"},
		},
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	err = subscriber.HandleEventBridge(context.Background(), awsevents.CloudWatchEvent{
		DetailType: contractevents.InboxMessageReceivedDetailType,
		Detail:     detail,
	})
	if err != nil {
		t.Fatalf("handle event failed: %v", err)
	}

	if router.routed != 1 {
		t.Fatalf("expected 1 routed event, got %d", router.routed)
	}
}
