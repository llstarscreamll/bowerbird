package domain

import (
	"testing"
	"time"
)

func TestInvoiceDocumentValidate(t *testing.T) {
	doc := &InvoiceDocument{
		CUFE:      "CUFE-1",
		InvoiceID: "INV-1",
		Issuer:    Party{Name: "Issuer", CompanyID: "900"},
		Receiver:  Party{Name: "Receiver", CompanyID: "901"},
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
