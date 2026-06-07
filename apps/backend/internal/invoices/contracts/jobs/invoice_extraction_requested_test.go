package jobs

import "testing"

func TestMarshalUnmarshalInvoiceExtractionRequested(t *testing.T) {
	payload, err := MarshalInvoiceExtractionRequested(InvoiceExtractionRequested{
		JobID:    "job_1",
		Source:   "files-uploaded-by-user",
		Files:    []File{{Path: "k1", Filename: "factura.xml"}},
		QueuedAt: "2026-06-03T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	decoded, err := UnmarshalInvoiceExtractionRequested(payload)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Source != "files-uploaded-by-user" {
		t.Fatalf("expected source_message_id files-uploaded-by-user, got %q", decoded.Source)
	}
	if len(decoded.Files) != 1 {
		t.Fatalf("expected one file, got %d", len(decoded.Files))
	}
}

func TestMarshalInvoiceExtractionRequestedMissingRequiredFields(t *testing.T) {
	_, err := MarshalInvoiceExtractionRequested(InvoiceExtractionRequested{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
