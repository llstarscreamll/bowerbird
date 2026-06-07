package ports

import (
	"context"

	"github.com/bowerbird/internal/invoices/domain"
)

type InvoiceXMLExtractor interface {
	ParseInvoiceXML(data []byte) (*domain.InvoiceDocument, error)
}

type InvoiceLLMExtractor interface {
	ExtractFromPDF(ctx context.Context, pdfData []byte) (*domain.InvoiceDocument, error)
}
