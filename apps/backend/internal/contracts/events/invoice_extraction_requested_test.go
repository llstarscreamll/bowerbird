package events

import (
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
)

func TestMarshalUnmarshalInvoiceExtractionRequested(t *testing.T) {
	payload, err := MarshalInvoiceExtractionRequested(InvoiceExtractionRequested{
		EventID:         "evt_1",
		OccurredAt:      "2026-06-03T12:00:00Z",
		TenantSlug:      "tenant-1",
		SourceMessageID: "msg-1",
		AttachmentRefs: []AttachmentRef{
			{S3Key: "k1", Filename: "factura.xml"},
		},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	decoded, err := UnmarshalInvoiceExtractionRequested(payload)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.SourceMessageID != "msg-1" {
		t.Fatalf("expected source_message_id msg-1, got %q", decoded.SourceMessageID)
	}
	if len(decoded.AttachmentRefs) != 1 {
		t.Fatalf("expected one attachment, got %d", len(decoded.AttachmentRefs))
	}
}

func TestMarshalInvoiceExtractionRequestedMissingRequiredFields(t *testing.T) {
	_, err := MarshalInvoiceExtractionRequested(InvoiceExtractionRequested{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDecodeInvoiceExtractionRequestedFromCloudWatchEvent(t *testing.T) {
	detail, err := MarshalInvoiceExtractionRequested(InvoiceExtractionRequested{
		EventID:         "evt_1",
		TenantSlug:      "tenant-1",
		SourceMessageID: "msg-1",
		AttachmentRefs: []AttachmentRef{
			{S3Key: "k1", Filename: "factura.pdf"},
		},
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	decoded, err := DecodeInvoiceExtractionRequestedFromCloudWatchEvent(awsevents.CloudWatchEvent{
		DetailType: InvoiceExtractionRequestedDetailType,
		Detail:     detail,
	})
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.TenantSlug != "tenant-1" {
		t.Fatalf("expected tenant slug tenant-1, got %q", decoded.TenantSlug)
	}
}

func TestDecodeInvoiceExtractionRequestedFromCloudWatchEventWrongType(t *testing.T) {
	_, err := DecodeInvoiceExtractionRequestedFromCloudWatchEvent(awsevents.CloudWatchEvent{
		DetailType: "OtherEvent",
		Detail:     []byte(`{"event_id":"evt_1"}`),
	})
	if err == nil {
		t.Fatal("expected detail type error")
	}
}
