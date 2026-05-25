package domain

import (
	"context"
	"time"
)

type InvoiceHeaderRecord struct {
	ID               string
	SourceMessageID  string
	CUFE             string
	InvoiceNumber    string
	IssuerName       string
	IssuerTaxID      string
	ReceiverName     string
	ReceiverTaxID    string
	CurrencyCode     string
	IssueDate        *time.Time
	DueDate          *time.Time
	PaymentCode      string
	Subtotal         float64
	TaxTotal         float64
	GrandTotal       float64
	DocumentRefS3Key string
	ExtractionSource string
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
	RawData         []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type InvoiceWriteRepository interface {
	PersistInvoiceAtomic(ctx context.Context, header InvoiceHeaderRecord, lines []InvoiceLineRecord) error
}
