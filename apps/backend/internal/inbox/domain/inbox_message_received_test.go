package domain

import (
	"encoding/json"
	"testing"

	awsevents "github.com/aws/aws-lambda-go/events"
)

func TestMarshalUnmarshalInboxMessageReceived(t *testing.T) {
	event := InboxMessageReceived{
		EventID:           "01JWEVENT1234567890ABCDEFG",
		OccurredAt:        "2026-05-25T12:00:00Z",
		TenantSlug:        "tenant-acme",
		AccountID:         "01JWACCOUNT123456789ABCDE",
		Provider:          "gmail",
		ProviderMessageID: "msg_123",
		MessageInternalID: "01JWMESSAGE123456789ABCDE",
		Subject:           "Factura mayo",
		Sender:            "billing@vendor.com",
		AttachmentRefs: []AttachmentRef{
			{
				S3Key:    "tenant/t_123/inbox/raw/2026/05/25/msg_1/factura.xml",
				Filename: "factura.xml",
				MimeType: "application/xml",
				SHA256:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}

	body, err := MarshalInboxMessageReceived(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	decoded, err := UnmarshalInboxMessageReceived(body)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.EventID != event.EventID {
		t.Fatalf("expected event id %q, got %q", event.EventID, decoded.EventID)
	}

	if decoded.TenantSlug != event.TenantSlug {
		t.Fatalf("expected tenant slug %q, got %q", event.TenantSlug, decoded.TenantSlug)
	}

	if decoded.ProviderMessageID != event.ProviderMessageID {
		t.Fatalf("expected provider message id %q, got %q", event.ProviderMessageID, decoded.ProviderMessageID)
	}

	if len(decoded.AttachmentRefs) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(decoded.AttachmentRefs))
	}
}

func TestMarshalInboxMessageReceivedMissingRequiredField(t *testing.T) {
	_, err := MarshalInboxMessageReceived(InboxMessageReceived{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDecodeInboxMessageReceivedFromCloudWatchEvent(t *testing.T) {
	detail, err := json.Marshal(InboxMessageReceived{
		EventID:           "01JWEVENT1234567890ABCDEFG",
		TenantSlug:        "tenant-acme",
		AccountID:         "01JWACCOUNT123456789ABCDE",
		Provider:          "gmail",
		ProviderMessageID: "msg_123",
		MessageInternalID: "01JWMESSAGE123456789ABCDE",
	})
	if err != nil {
		t.Fatalf("marshal detail failed: %v", err)
	}

	cloudwatchEvent := awsevents.CloudWatchEvent{
		ID:         "evt_1",
		DetailType: InboxMessageReceivedDetailType,
		Source:     InboxMessageReceivedSource,
		Detail:     detail,
	}

	decoded, err := DecodeInboxMessageReceivedFromCloudWatchEvent(cloudwatchEvent)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.MessageInternalID != "01JWMESSAGE123456789ABCDE" {
		t.Fatalf("expected message_internal_id to match, got %q", decoded.MessageInternalID)
	}
}

func TestDecodeInboxMessageReceivedFromCloudWatchEventWrongType(t *testing.T) {
	cloudwatchEvent := awsevents.CloudWatchEvent{
		DetailType: "OtherEvent",
		Detail:     []byte(`{"event_id":"1"}`),
	}

	_, err := DecodeInboxMessageReceivedFromCloudWatchEvent(cloudwatchEvent)
	if err == nil {
		t.Fatal("expected detail type error")
	}
}
