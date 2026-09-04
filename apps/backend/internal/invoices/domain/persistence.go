package domain

import (
	"time"
)

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
