package events

import (
	"context"
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingapp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	platformevents "github.com/money-path/bowerbird/apps/backend/internal/platform/events"
)

type fakePublisher struct {
	published int
}

func (p *fakePublisher) PublishBusinessEvent(ctx context.Context, event platformevents.BusinessEvent) error {
	p.published++
	return nil
}

func TestSubscriberRoutesInboxMessageReceivedEvent(t *testing.T) {
	publisher := &fakePublisher{}
	cmd := invoicingapp.NewCheckInboxMessageForInvoiceCandidatesCommand(publisher)
	subscriber := NewInboxMessageReceivedSubscriber(cmd)

	detail, err := contractevents.MarshalInboxMessageReceived(contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantSlug:        "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica",
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

	if publisher.published != 1 {
		t.Fatalf("expected 1 published event, got %d", publisher.published)
	}
}
