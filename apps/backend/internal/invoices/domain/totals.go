package domain

// InvoiceTotals is the legal monetary total of an invoice (value object).
type InvoiceTotals struct {
	Subtotal       float64
	TaxTotal       float64
	AllowanceTotal float64
	GrandTotal     float64
}

func (t InvoiceTotals) HasAllowance() bool {
	return t.AllowanceTotal > 0
}

func (d *InvoiceDocument) Totals() InvoiceTotals {
	if d == nil {
		return InvoiceTotals{}
	}
	return InvoiceTotals{
		Subtotal:       d.LineExtension,
		TaxTotal:       d.TaxAmountTotal(),
		AllowanceTotal: d.AllowanceTotal,
		GrandTotal:     d.PayableAmount,
	}
}
