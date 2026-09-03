package v1

import (
	"encoding/json"
	"time"

	"github.com/bowerbird/internal/invoices/application/commands"
	"github.com/bowerbird/internal/invoices/application/queries"
	"github.com/bowerbird/internal/invoices/domain"
)

type jsonApiResponse[T any] struct {
	Data jsonApiDocument[T] `json:"data"`
}

type jsonApiCollectionResponse[T any] struct {
	Data []jsonApiDocument[T]  `json:"data"`
	Meta jsonApiCollectionMeta `json:"meta,omitempty"`
}

type jsonApiCollectionMeta struct {
	TotalCount *int    `json:"total_count,omitempty"`
	Limit      int     `json:"limit"`
	Offset     *int    `json:"offset,omitempty"`
	Cursor     *string `json:"cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type jsonApiDocument[T any] struct {
	Type       string `json:"type,omitempty"`
	ID         string `json:"id,omitempty"`
	Attributes T      `json:"attributes,omitempty"`
}

type queueInvoiceExtractionResponse struct {
	JobID            string `json:"job_id"`
	Status           string `json:"status"`
	QueuedFilesCount int    `json:"queued_files_count"`
}

func newQueueInvoiceExtractionResponse(result *commands.QueueInvoiceExtractionFromFilesResult) jsonApiResponse[queueInvoiceExtractionResponse] {
	return jsonApiResponse[queueInvoiceExtractionResponse]{
		Data: jsonApiDocument[queueInvoiceExtractionResponse]{
			Type: "queue-invoice-extraction",
			ID:   result.JobID,
			Attributes: queueInvoiceExtractionResponse{
				JobID:            result.JobID,
				Status:           "queued",
				QueuedFilesCount: result.QueuedFilesCount,
			},
		},
	}
}

type invoiceSummaryAttributes struct {
	SourceName       string  `json:"source_name"`
	SourceID         string  `json:"source_id"`
	CUFE             string  `json:"cufe"`
	InvoiceNumber    string  `json:"invoice_number"`
	IssuerName       string  `json:"issuer_name"`
	IssuerTaxID      string  `json:"issuer_tax_id"`
	IssuerPartyID    *string `json:"issuer_party_id"`
	ReceiverName     string  `json:"receiver_name"`
	ReceiverTaxID    string  `json:"receiver_tax_id"`
	CurrencyCode     string  `json:"currency_code"`
	IssueDate        *string `json:"issue_date"`
	DueDate          *string `json:"due_date"`
	PaymentCode      string  `json:"payment_code"`
	Subtotal         float64 `json:"subtotal"`
	TaxTotal         float64 `json:"tax_total"`
	AllowanceTotal   float64 `json:"allowance_total"`
	GrandTotal       float64 `json:"grand_total"`
	ExtractionSource string  `json:"extraction_source"`
	LinkingStatus    string  `json:"linking_status"`
	CreatedAt        string  `json:"created_at"`
}

type invoiceLineAttributes struct {
	ID           string  `json:"id"`
	LineNumber   int     `json:"line_number"`
	ItemCode     string  `json:"item_code"`
	Description  string  `json:"description"`
	Quantity     float64 `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	LineTaxTotal float64 `json:"line_tax_total"`
	LineTotal    float64 `json:"line_total"`
	ItemID       *string `json:"item_id"`
	LinkStatus   string  `json:"link_status"`
	LinkMethod   *string `json:"link_method"`
	LinkLocked   bool    `json:"link_locked"`
	Suggestions  any     `json:"suggestions"`
}

type invoiceDetailsAttributes struct {
	invoiceSummaryAttributes
	Lines []invoiceLineAttributes `json:"lines"`
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

func toInvoiceSummaryAttributes(header domain.InvoiceHeaderRecord) invoiceSummaryAttributes {
	return invoiceSummaryAttributes{
		SourceName:       header.SourceName,
		SourceID:         header.SourceID,
		CUFE:             header.CUFE,
		InvoiceNumber:    header.InvoiceNumber,
		IssuerName:       header.IssuerName,
		IssuerTaxID:      header.IssuerTaxID,
		IssuerPartyID:    optionalString(header.IssuerPartyID),
		ReceiverName:     header.ReceiverName,
		ReceiverTaxID:    header.ReceiverTaxID,
		CurrencyCode:     header.CurrencyCode,
		IssueDate:        formatTimePtr(header.IssueDate),
		DueDate:          formatTimePtr(header.DueDate),
		PaymentCode:      header.PaymentCode,
		Subtotal:         header.Subtotal,
		TaxTotal:         header.TaxTotal,
		AllowanceTotal:   header.AllowanceTotal,
		GrandTotal:       header.GrandTotal,
		ExtractionSource: header.ExtractionSource,
		LinkingStatus:    header.LinkingStatus,
		CreatedAt:        header.CreatedAt.Format(time.RFC3339),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func newInvoiceListResponse(result *queries.PaginatedInvoices) jsonApiCollectionResponse[invoiceSummaryAttributes] {
	docs := make([]jsonApiDocument[invoiceSummaryAttributes], 0, len(result.Items))
	for _, item := range result.Items {
		docs = append(docs, jsonApiDocument[invoiceSummaryAttributes]{
			Type:       "invoices",
			ID:         item.ID,
			Attributes: toInvoiceSummaryAttributes(item),
		})
	}

	var cursor *string
	if result.Cursor != "" {
		cursor = &result.Cursor
	}

	return jsonApiCollectionResponse[invoiceSummaryAttributes]{
		Data: docs,
		Meta: jsonApiCollectionMeta{
			Limit:   result.Limit,
			Cursor:  cursor,
			HasMore: result.HasMore,
		},
	}
}

func newInvoiceDetailsResponse(result *queries.InvoiceDetails) jsonApiResponse[invoiceDetailsAttributes] {
	lines := make([]invoiceLineAttributes, 0, len(result.Lines))
	for _, line := range result.Lines {
		var suggestions any = []any{}
		if len(line.Suggestions) > 0 {
			_ = json.Unmarshal(line.Suggestions, &suggestions)
		}
		lines = append(lines, invoiceLineAttributes{
			ID:           line.ID,
			LineNumber:   line.LineNumber,
			ItemCode:     line.ItemCode,
			Description:  line.Description,
			Quantity:     line.Quantity,
			UnitPrice:    line.UnitPrice,
			LineTaxTotal: line.LineTaxTotal,
			LineTotal:    line.LineTotal,
			ItemID:       optionalString(line.ItemID),
			LinkStatus:   line.LinkStatus,
			LinkMethod:   optionalString(line.LinkMethod),
			LinkLocked:   line.LinkLocked,
			Suggestions:  suggestions,
		})
	}

	return jsonApiResponse[invoiceDetailsAttributes]{
		Data: jsonApiDocument[invoiceDetailsAttributes]{
			Type: "invoices",
			ID:   result.Header.ID,
			Attributes: invoiceDetailsAttributes{
				invoiceSummaryAttributes: toInvoiceSummaryAttributes(result.Header),
				Lines:                    lines,
			},
		},
	}
}
