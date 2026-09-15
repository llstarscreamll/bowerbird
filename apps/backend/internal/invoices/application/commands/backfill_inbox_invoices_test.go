package commands

import (
	"context"
	"testing"

	inboxapi "github.com/bowerbird/internal/inbox/api"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type backfillSourceStub struct {
	page inboxapi.ExtractionCandidatePage
}

func (s backfillSourceStub) ListExtractionCandidates(ctx context.Context, cursor string, limit int) (inboxapi.ExtractionCandidatePage, error) {
	return s.page, nil
}

type backfillQueueSpy struct {
	jobs []jobs.Job
}

func (p *backfillQueueSpy) Enqueue(ctx context.Context, job jobs.Job) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestBackfillInboxInvoicesEnqueuesCandidates(t *testing.T) {
	queue := &backfillQueueSpy{}
	enqueue := NewCreateInvoicesFromInboxMessageCommand(queue, matchingReceivers())
	cmd := NewBackfillInboxInvoicesCommand(
		backfillSourceStub{page: inboxapi.ExtractionCandidatePage{
			Items: []inboxapi.ExtractionCandidate{{
				MessageID: "m-1",
				Subject:   "Factura electronica",
				Attachments: []inboxapi.AttachmentRef{
					{S3Key: "k1", Filename: "factura.xml", MimeType: "application/xml"},
				},
			}},
		}},
		enqueue,
		queue,
		matchingReceivers(),
	)
	ctx := tenant.WithTenantID(context.Background(), "tenant_1")
	err := cmd.Execute(ctx, contractJobs.InboxBackfillJob{ID: "job-1"})
	require.NoError(t, err)
	require.Len(t, queue.jobs, 1)
	assert.Equal(t, contractJobs.InvoiceExtractionRequestedType, queue.jobs[0].Type)
}

func TestBackfillInboxInvoicesEnqueueSkipsWithoutLegalEntity(t *testing.T) {
	queue := &backfillQueueSpy{}
	enqueue := NewCreateInvoicesFromInboxMessageCommand(queue, matchingReceivers())
	cmd := NewBackfillInboxInvoicesCommand(backfillSourceStub{}, enqueue, queue, receiverDirectoryStub{})
	err := cmd.Enqueue(context.Background())
	require.NoError(t, err)
	assert.Empty(t, queue.jobs)
}
