package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMalformedXML     = errors.New("malformed xml invoice")
	ErrMissingCUFE      = errors.New("missing cufe/uuid")
	ErrMissingIssuer    = errors.New("missing issuer data")
	ErrMissingReceiver  = errors.New("missing receiver data")
	ErrMissingLineItems = errors.New("missing invoice line items")
	ErrMissingInvoiceID = errors.New("missing invoice id")
)

type Party struct {
	Name           string
	CompanyID      string
	SchemeID       string
	TaxLevelCode   string
	RegistrationID string
}

type TaxTotal struct {
	TaxAmount float64
	Taxable   float64
	TaxCode   string
	Percent   float64
}

type InvoiceLine struct {
	LineID          string
	ItemDescription string
	Quantity        float64
	UnitCode        string
	UnitPrice       float64
	LineExtension   float64
	TaxAmount       float64
}

type InvoiceDocument struct {
	ProfileID        string
	InvoiceID        string
	IssueDate        string
	IssueTime        string
	CurrencyCode     string
	CUFE             string
	PaymentMeansCode string
	Issuer           Party
	Receiver         Party
	TaxTotals        []TaxTotal
	LineExtension    float64
	TaxExclusive     float64
	TaxInclusive     float64
	PayableAmount    float64
	Lines            []InvoiceLine
	RawData          []byte
}

func (d *InvoiceDocument) Validate() error {
	if d == nil {
		return ErrMalformedXML
	}
	if strings.TrimSpace(d.CUFE) == "" {
		return ErrMissingCUFE
	}
	if strings.TrimSpace(d.InvoiceID) == "" {
		return ErrMissingInvoiceID
	}
	if strings.TrimSpace(d.Issuer.Name) == "" || strings.TrimSpace(d.Issuer.CompanyID) == "" {
		return ErrMissingIssuer
	}
	if strings.TrimSpace(d.Receiver.Name) == "" || strings.TrimSpace(d.Receiver.CompanyID) == "" {
		return ErrMissingReceiver
	}
	if len(d.Lines) == 0 {
		return ErrMissingLineItems
	}

	return nil
}

func (d *InvoiceDocument) TaxAmountTotal() float64 {
	total := 0.0
	for _, tax := range d.TaxTotals {
		total += tax.TaxAmount
	}
	return total
}

func (d *InvoiceDocument) IssueDateTimeUTC() *time.Time {
	date := strings.TrimSpace(d.IssueDate)
	if date == "" {
		return nil
	}

	timePart := strings.TrimSpace(d.IssueTime)
	if timePart == "" {
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil
		}
		utc := parsed.UTC()
		return &utc
	}

	combined := date + "T" + timePart
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, combined)
		if err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}

	return nil
}

func (l InvoiceLine) NumberOrDefault(fallback int) int {
	trimmed := strings.TrimSpace(l.LineID)
	if trimmed == "" {
		return fallback
	}
	number, err := strconv.Atoi(trimmed)
	if err != nil || number <= 0 {
		return fallback
	}

	return number
}
