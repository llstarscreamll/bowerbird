package domain

import "context"

type InvoiceXMLExtractor interface {
	ParseInvoiceXML(data []byte) (*InvoiceDocument, error)
}

type InvoiceLLMExtractor interface {
	ExtractFromPDF(ctx context.Context, pdfData []byte) (*InvoiceDocument, error)
}
