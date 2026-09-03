package domain

import (
	"testing"
	"time"
)

func TestInvoiceDocumentValidate(t *testing.T) {
	doc := &InvoiceDocument{
		CUFE:      "CUFE-1",
		InvoiceID: "INV-1",
		Issuer:    Party{Name: "Issuer", TaxID: "900"},
		Receiver:  Party{Name: "Receiver", TaxID: "901"},
		Lines:     []InvoiceLine{{LineID: "1", ItemDescription: "Service"}},
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("expected valid document, got %v", err)
	}
}

func TestInvoiceDocumentTaxAmountTotal(t *testing.T) {
	doc := &InvoiceDocument{TaxTotals: []TaxTotal{{TaxAmount: 10}, {TaxAmount: 9.5}}}

	if got := doc.TaxAmountTotal(); got != 19.5 {
		t.Fatalf("expected tax total 19.5, got %f", got)
	}
}

func TestInvoiceDocumentIssueDateTimeUTC(t *testing.T) {
	doc := &InvoiceDocument{IssueDate: "2026-05-25", IssueTime: "10:00:00-05:00"}

	got := doc.IssueDateTimeUTC()
	if got == nil {
		t.Fatal("expected parsed datetime")
	}

	want := time.Date(2026, 5, 25, 15, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestInvoiceDocumentDueDateTimeUTC(t *testing.T) {
	doc := &InvoiceDocument{DueDate: "2026-08-08"}

	got := doc.DueDateTimeUTC()
	if got == nil {
		t.Fatal("expected parsed due date")
	}

	want := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestResolveDueDatePrefersInvoiceDueDate(t *testing.T) {
	if got := ResolveDueDate("2026-08-08", "2026-08-06"); got != "2026-08-08" {
		t.Fatalf("expected invoice due date, got %q", got)
	}
	if got := ResolveDueDate("", "2026-08-06"); got != "2026-08-06" {
		t.Fatalf("expected payment due date fallback, got %q", got)
	}
}

func TestInvoiceDocumentTotalsMapsPayableAndAllowance(t *testing.T) {
	doc := &InvoiceDocument{
		LineExtension:  2628000,
		AllowanceTotal: 2618000,
		PayableAmount:  10000,
		TaxTotals:      []TaxTotal{{TaxAmount: 0}},
	}

	got := doc.Totals()
	if got.Subtotal != 2628000 || got.TaxTotal != 0 || got.AllowanceTotal != 2618000 || got.GrandTotal != 10000 {
		t.Fatalf("unexpected totals: %#v", got)
	}
	if !got.HasAllowance() {
		t.Fatal("expected allowance")
	}
}

func TestInvoiceLineNumberOrDefault(t *testing.T) {
	line := InvoiceLine{LineID: "3"}
	if got := line.NumberOrDefault(1); got != 3 {
		t.Fatalf("expected line number 3, got %d", got)
	}

	invalid := InvoiceLine{LineID: "x"}
	if got := invalid.NumberOrDefault(7); got != 7 {
		t.Fatalf("expected fallback number 7, got %d", got)
	}
}
