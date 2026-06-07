package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type requestInvoiceExtractionPublisherSpy struct {
	jobs []jobs.Job
}

func (p *requestInvoiceExtractionPublisherSpy) Dispatch(ctx context.Context, job jobs.Job) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestQueueInvoiceExtractionFromUploadedFilesCommandQueuesJob(t *testing.T) {
	publisher := &requestInvoiceExtractionPublisherSpy{}
	cmd := NewQueueInvoiceExtractionFromFilesCommand(publisher)
	cmd.newID = func() string { return "evt_123" }
	cmd.now = func() time.Time { return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC) }
	ctx := context.Background()
	ctx = tenant.WithTenantID(ctx, "tenant_1")

	result, err := cmd.Execute(ctx, QueueInvoiceExtractionFromFilesInput{
		Files: []File{
			{Name: "invoice-a.PDF", Path: "uploads/invoicing/user-a/invoice-a.pdf", MimeType: "PDF"},
			{Name: "invoice-b.xml", Path: "uploads/invoicing/user-a/invoice-b.xml", MimeType: "xml"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "evt_123", result.JobID)
	assert.Equal(t, 2, result.QueuedFilesCount)
	require.Len(t, publisher.jobs, 1)
	assert.Equal(t, contractJobs.InvoiceExtractionRequestedType, publisher.jobs[0].Type)

	var queued contractJobs.InvoiceExtractionRequested
	require.NoError(t, json.Unmarshal(publisher.jobs[0].Payload, &queued))
	assert.Equal(t, "files-uploaded-by-user", queued.Source)
	require.Len(t, queued.Files, 2)
	assert.Equal(t, "PDF", queued.Files[0].MimeType)
}
