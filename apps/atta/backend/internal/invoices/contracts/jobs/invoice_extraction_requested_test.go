package jobs

import "testing"

func TestMarshalUnmarshalInvoiceExtractionRequested(t *testing.T) {
	payload, err := MarshalInvoiceExtractionRequested(ExtractInvoicesFromFilesJob{
		ID:         "job_1",
		SourceName: "files-uploaded-by-user",
		SourceID:   "upload_batch_1",
		Files:      []File{{Path: "k1", Filename: "factura.xml"}},
		QueuedAt:   "2026-06-03T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	decoded, err := UnmarshalInvoiceExtractionRequested(payload)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.SourceName != "files-uploaded-by-user" {
		t.Fatalf("expected source_name files-uploaded-by-user, got %q", decoded.SourceName)
	}
	if decoded.SourceID != "upload_batch_1" {
		t.Fatalf("expected source_id upload_batch_1, got %q", decoded.SourceID)
	}
	if len(decoded.Files) != 1 {
		t.Fatalf("expected one file, got %d", len(decoded.Files))
	}
}

func TestMarshalInvoiceExtractionRequestedMissingRequiredFields(t *testing.T) {
	_, err := MarshalInvoiceExtractionRequested(ExtractInvoicesFromFilesJob{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
