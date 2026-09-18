package v1

import "testing"

const testULID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func TestQueueInvoiceExtractionRequestDocumentValidateAcceptsTextXML(t *testing.T) {
	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
			ID:   testULID,
			Type: queueInvoiceExtractionDataType,
			Attributes: queueInvoiceExtractionRequestDataAttrs{
				Files: []file{
					{Name: "invoice.xml", Path: "uploads/invoicing/u1/invoice.xml", MimeType: "text/xml"},
				},
			},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}
