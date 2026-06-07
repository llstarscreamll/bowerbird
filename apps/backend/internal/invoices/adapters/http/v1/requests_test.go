package v1

import "testing"

func TestQueueInvoiceExtractionRequestDocumentValidateSuccess(t *testing.T) {
	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
			Type: queueInvoiceExtractionDataType,
			Attributes: queueInvoiceExtractionRequestDataAttrs{
				Files: []file{
					{Name: "invoice.pdf", Path: "uploads/invoicing/u1/invoice.pdf", MimeType: "application/pdf"},
				},
			},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestQueueInvoiceExtractionRequestDocumentValidateInvalidType(t *testing.T) {
	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
			Type: queueInvoiceExtractionDataType,
			Attributes: queueInvoiceExtractionRequestDataAttrs{
				Files: []file{
					{Name: "invoice.txt", Path: "uploads/invoicing/u1/invoice.txt", MimeType: "text/plain"},
				},
			},
		},
	}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestQueueInvoiceExtractionRequestDocumentValidateTooManyFiles(t *testing.T) {
	files := make([]file, 51)
	for i := range files {
		files[i] = file{Name: "invoice.pdf", Path: "uploads/invoicing/u1/invoice.pdf", MimeType: "application/pdf"}
	}

	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
			Type: queueInvoiceExtractionDataType,
			Attributes: queueInvoiceExtractionRequestDataAttrs{
				Files: files,
			},
		},
	}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestQueueInvoiceExtractionRequestDocumentValidateMimeAndExtensionMismatch(t *testing.T) {
	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
			Type: queueInvoiceExtractionDataType,
			Attributes: queueInvoiceExtractionRequestDataAttrs{
				Files: []file{
					{Name: "invoice.xml", Path: "uploads/invoicing/u1/invoice.xml", MimeType: "application/pdf"},
				},
			},
		},
	}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestQueueInvoiceExtractionRequestDocumentValidateAcceptsTextXML(t *testing.T) {
	req := queueInvoiceExtractionRequestDocument{
		Data: jsonApiDocument[queueInvoiceExtractionRequestDataAttrs]{
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
