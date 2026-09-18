package storage

import "testing"

func TestSanitizeObjectMetadataValueMacOSScreenshot(t *testing.T) {
	in := "Screenshot 2026-07-08 at 8.12.01\u202fAM.png"
	got := SanitizeObjectMetadataValue(in)
	want := "Screenshot 2026-07-08 at 8.12.01 AM.png"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSanitizeObjectMetadataValueDegreeSymbol(t *testing.T) {
	in := "Factura N° 123.pdf"
	got := SanitizeObjectMetadataValue(in)
	want := "Factura N_ 123.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMetadataToPresignHeaders(t *testing.T) {
	got := MetadataToPresignHeaders(map[string]string{
		"tenant-id": "t1",
		"module":    "invoices",
	})
	if got["x-amz-meta-tenant-id"] != "t1" {
		t.Fatalf("unexpected tenant header: %#v", got)
	}
	if got["x-amz-meta-module"] != "invoices" {
		t.Fatalf("unexpected module header: %#v", got)
	}
}
