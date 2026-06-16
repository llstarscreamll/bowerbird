package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	contractevents "github.com/bowerbird/internal/contracts/events"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBusinessPublisher struct {
	jobs []jobs.Job
}

func (p *fakeBusinessPublisher) Dispatch(ctx context.Context, job jobs.Job) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestCheckQueuesInvoiceExtractionJob(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCreateInvoicesFromInboxMessageCommand(publisher)
	uc.newID = func() string { return "evt_1" }

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_src_1",
		TenantID:          "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "provider_msg_1",
		MessageInternalID: "m_1",
		Subject:           "Factura electronica de mayo",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "factura.pdf"},
		},
	})
	require.NoError(t, err)
	require.Len(t, publisher.jobs, 1)
	assert.Equal(t, contractJobs.InvoiceExtractionRequestedType, publisher.jobs[0].Type)

	var queued contractJobs.InvoiceExtractionRequested
	require.NoError(t, json.Unmarshal(publisher.jobs[0].Payload, &queued))
	assert.Equal(t, "inbox-message", queued.Source)
}

func TestCheckSkipsNonCandidates(t *testing.T) {
	publisher := &fakeBusinessPublisher{}
	uc := NewCreateInvoicesFromInboxMessageCommand(publisher)

	err := uc.Execute(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt_1",
		TenantID:          "tenant_1",
		AccountID:         "acc_1",
		Provider:          "gmail",
		ProviderMessageID: "msg_1",
		MessageInternalID: "m_1",
		Subject:           "meeting notes from vendor",
		AttachmentRefs: []contractevents.AttachmentRef{
			{Filename: "notes.txt"},
		},
	})
	require.NoError(t, err)
	assert.Len(t, publisher.jobs, 0)
}

type fakeInvoiceWriteRepo struct {
	called bool
	header domain.InvoiceHeaderRecord
	lines  []domain.InvoiceLineRecord
}

func (r *fakeInvoiceWriteRepo) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	r.called = true
	r.header = header
	r.lines = lines
	return nil
}

func TestCreateInvoiceCommandBuildsAtomicRecords(t *testing.T) {
	repo := &fakeInvoiceWriteRepo{}
	uc := NewCreateInvoiceCommand(repo)
	uc.now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	ids := []string{"hdr_1", "line_1", "line_2"}
	i := 0
	uc.newID = func() string {
		id := ids[i]
		i++
		return id
	}

	res, err := uc.Execute(context.Background(), CreateInvoiceInput{
		SourceMessageID:  "msg_1",
		ExtractionSource: "xml",
		DocumentRefS3Key: "tenant/t/inbox/.../invoice.xml",
		Invoice: &domain.InvoiceDocument{
			CUFE:             "CUFE-1",
			InvoiceID:        "FE-1",
			IssueDate:        "2026-05-25",
			IssueTime:        "10:00:00-05:00",
			CurrencyCode:     "COP",
			PaymentMeansCode: "1",
			Issuer:           domain.Party{Name: "Proveedor", CompanyID: "900"},
			Receiver:         domain.Party{Name: "Cliente", CompanyID: "901"},
			LineExtension:    100,
			TaxTotals:        []domain.TaxTotal{{TaxAmount: 19}, {TaxAmount: 1}},
			PayableAmount:    120,
			RawData:          []byte(`{"src":"xml"}`),
			Lines: []domain.InvoiceLine{
				{LineID: "1", ItemDescription: "Servicio A", Quantity: 1, UnitPrice: 50, LineExtension: 50, TaxAmount: 9.5},
				{LineID: "2", ItemDescription: "Servicio B", Quantity: 1, UnitPrice: 50, LineExtension: 50, TaxAmount: 10.5},
			},
		},
	})
	require.NoError(t, err)
	assert.True(t, repo.called)
	assert.Equal(t, "CUFE-1", repo.header.CUFE)
	assert.Equal(t, 20.0, repo.header.TaxTotal)
	require.Len(t, repo.lines, 2)
	assert.Equal(t, 1, repo.lines[0].LineNumber)
	assert.Equal(t, 2, repo.lines[1].LineNumber)
	assert.Equal(t, "hdr_1", res.HeaderID)
	assert.Len(t, res.LineIDs, 2)
}
