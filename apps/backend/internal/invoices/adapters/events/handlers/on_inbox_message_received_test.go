package handlers

import (
	"context"
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/bowerbird/internal/contracts/events"
	invoicingcommands "github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/platform/jobs"
)

type fakePublisher struct {
	enqueued int
}

func (p *fakePublisher) Dispatch(ctx context.Context, job jobs.Job) error {
	p.enqueued++
	return nil
}

func TestOnInboxMessageReceivedRoutesEvent(t *testing.T) {
	publisher := &fakePublisher{}
	cmd := invoicingcommands.NewCreateInvoicesFromInboxMessageCommand(publisher)
	handler := NewOnInboxMessageReceived(cmd)

	detail, err := contractevents.MarshalInboxMessageReceived(contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantID:          "tenant_1",
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

	err = handler.HandleEventBridge(context.Background(), awsevents.CloudWatchEvent{
		DetailType: contractevents.InboxMessageReceivedDetailType,
		Detail:     detail,
	})
	if err != nil {
		t.Fatalf("handle event failed: %v", err)
	}

	if publisher.enqueued != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", publisher.enqueued)
	}
}
