package domain

import (
	"time"
)

// InvoiceReceipt is the capture identity for a received invoice (not XML content).
type InvoiceReceipt struct {
	ID               string
	SourceName       string
	SourceID         string
	ExtractionSource string
	StorageKey       string
	RawData          []byte
	ReceivedAt       time.Time
}

func (d *InvoiceDocument) ToHeaderRecord(receipt InvoiceReceipt) InvoiceHeaderRecord {
	totals := d.Totals()
	now := receipt.ReceivedAt.UTC()
	return InvoiceHeaderRecord{
		ID:               receipt.ID,
		SourceName:       receipt.SourceName,
		SourceID:         receipt.SourceID,
		CUFE:             d.CUFE,
		InvoiceNumber:    d.InvoiceID,
		IssuerName:       d.Issuer.Name,
		IssuerTaxID:      d.Issuer.TaxID,
		ReceiverName:     d.Receiver.Name,
		ReceiverTaxID:    d.Receiver.TaxID,
		CurrencyCode:     d.CurrencyCode,
		IssueDate:        d.IssueDateTimeUTC(),
		DueDate:          d.DueDateTimeUTC(),
		PaymentCode:      d.PaymentMeansCode,
		Subtotal:         totals.Subtotal,
		TaxTotal:         totals.TaxTotal,
		AllowanceTotal:   totals.AllowanceTotal,
		GrandTotal:       totals.GrandTotal,
		DocumentRefS3Key: receipt.StorageKey,
		ExtractionSource: receipt.ExtractionSource,
		LinkingStatus:    LinkingStatusPending,
		RawData:          receipt.RawData,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (l InvoiceLine) ToLineRecord(id, headerID string, fallbackNumber int, raw []byte, now time.Time) InvoiceLineRecord {
	now = now.UTC()
	return InvoiceLineRecord{
		ID:              id,
		InvoiceHeaderID: headerID,
		LineNumber:      l.NumberOrDefault(fallbackNumber),
		ItemCode:        l.ItemCode,
		Description:     l.ItemDescription,
		Quantity:        l.Quantity,
		UnitPrice:       l.UnitPrice,
		LineTaxTotal:    l.TaxAmount,
		LineTotal:       l.LineExtension,
		LinkStatus:      LinkStatusUnmatched,
		RawData:         raw,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

type InvoiceHeaderRecord struct {
	ID               string
	SourceName       string
	SourceID         string
	CUFE             string
	InvoiceNumber    string
	IssuerName       string
	IssuerTaxID      string
	IssuerPartyID    string
	ReceiverName     string
	ReceiverTaxID    string
	CurrencyCode     string
	IssueDate        *time.Time
	DueDate          *time.Time
	PaymentCode      string
	Subtotal         float64
	TaxTotal         float64
	AllowanceTotal   float64
	GrandTotal       float64
	DocumentRefS3Key string
	ExtractionSource string
	LinkingStatus    string
	RawData          []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type InvoiceLineRecord struct {
	ID              string
	InvoiceHeaderID string
	LineNumber      int
	ItemCode        string
	Description     string
	Quantity        float64
	UnitPrice       float64
	LineTaxTotal    float64
	LineTotal       float64
	ItemID          string
	LinkStatus      string
	LinkMethod      string
	LinkLocked      bool
	Suggestions     []byte
	RawData         []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	// Display-only enrichment from catalog (not persisted).
	ItemName string
	ItemSKU  string
}
