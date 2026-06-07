package v1

import "testing"

func TestRequestUploadURLRequestValidateSuccess(t *testing.T) {
	req := requestUploadURLRequest{
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Module:      "invoices",
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestRequestUploadURLRequestValidateMissingFields(t *testing.T) {
	tests := []struct {
		name string
		req  requestUploadURLRequest
	}{
		{
			name: "missing filename",
			req: requestUploadURLRequest{
				ContentType: "application/pdf",
				Module:      "invoices",
			},
		},
		{
			name: "missing content_type",
			req: requestUploadURLRequest{
				Filename: "invoice.pdf",
				Module:   "invoices",
			},
		},
		{
			name: "missing module",
			req: requestUploadURLRequest{
				Filename:    "invoice.pdf",
				ContentType: "application/pdf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.Validate(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestRequestDownloadURLRequestValidateSuccess(t *testing.T) {
	req := requestDownloadURLRequest{Key: "1-day/tenant-1/uploads/invoices/u1/file.pdf"}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestRequestDownloadURLRequestValidateMissingKey(t *testing.T) {
	req := requestDownloadURLRequest{}

	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
