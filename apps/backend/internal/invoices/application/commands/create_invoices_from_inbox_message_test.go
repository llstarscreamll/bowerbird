package commands

import (
	"context"
	"testing"

	contractEvents "github.com/bowerbird/internal/contracts/events"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inboxEnqueueSpy struct {
	jobs []jobs.Job
}

func (p *inboxEnqueueSpy) Enqueue(ctx context.Context, job jobs.Job) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestCreateInvoicesFromInboxMessageSkipsWithoutLegalEntity(t *testing.T) {
	publisher := &inboxEnqueueSpy{}
	cmd := NewCreateInvoicesFromInboxMessageCommand(publisher, receiverDirectoryStub{})
	err := cmd.Execute(context.Background(), contractEvents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantID:          "tenant_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica",
		AttachmentRefs:    []contractEvents.AttachmentRef{{Filename: "factura.xml"}},
	})
	require.NoError(t, err)
	assert.Empty(t, publisher.jobs)
}

func TestCreateInvoicesFromInboxMessageQueuesWhenLegalEntityExists(t *testing.T) {
	publisher := &inboxEnqueueSpy{}
	cmd := NewCreateInvoicesFromInboxMessageCommand(publisher, matchingReceivers())
	err := cmd.Execute(context.Background(), contractEvents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantID:          "tenant_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica",
		AttachmentRefs:    []contractEvents.AttachmentRef{{Filename: "factura.xml"}},
	})
	require.NoError(t, err)
	require.Len(t, publisher.jobs, 1)
}
