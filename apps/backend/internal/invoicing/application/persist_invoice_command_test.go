package application

import (
	"context"
	"testing"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/invoicing/domain"
)

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

func TestPersistInvoiceUseCasePersistBuildsAtomicRecords(t *testing.T) {
	repo := &fakeInvoiceWriteRepo{}
	uc := NewPersistInvoiceCommand(repo)
	uc.now = func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) }
	ids := []string{"hdr_1", "line_1", "line_2"}
	i := 0
	uc.newID = func() string {
		id := ids[i]
		i++
		return id
	}

	res, err := uc.Execute(context.Background(), PersistInvoiceInput{
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
	if err != nil {
		t.Fatalf("persist failed: %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository persist call")
	}
	if repo.header.CUFE != "CUFE-1" || repo.header.TaxTotal != 20 {
		t.Fatalf("unexpected header mapping: %#v", repo.header)
	}
	if len(repo.lines) != 2 || repo.lines[0].LineNumber != 1 || repo.lines[1].LineNumber != 2 {
		t.Fatalf("unexpected lines mapping: %#v", repo.lines)
	}
	if res.HeaderID != "hdr_1" || len(res.LineIDs) != 2 {
		t.Fatalf("unexpected result ids: %#v", res)
	}
}
