package events

import (
	"encoding/json"
	"errors"
	"fmt"

	awsevents "github.com/aws/aws-lambda-go/events"
)

const (
	InvoiceExtractionRequestedDetailType    = "InvoiceExtractionRequested"
	InvoiceExtractionRequestedSource        = "bowerbird.invoicing"
	InvoiceExtractionRequestedSchemaVersion = "1.0"
)

type InvoiceExtractionRequested struct {
	EventID           string          `json:"event_id"`
	OccurredAt        string          `json:"occurred_at"`
	TenantSlug        string          `json:"tenant_slug"`
	SourceMessageID   string          `json:"source_message_id"`
	ProviderMessageID string          `json:"provider_message_id"`
	AttachmentRefs    []AttachmentRef `json:"attachment_refs"`
}

func (e InvoiceExtractionRequested) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if e.TenantSlug == "" {
		return errors.New("tenant_slug is required")
	}
	if e.SourceMessageID == "" {
		return errors.New("source_message_id is required")
	}
	if len(e.AttachmentRefs) == 0 {
		return errors.New("attachment_refs is required")
	}

	return nil
}

func MarshalInvoiceExtractionRequested(event InvoiceExtractionRequested) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(event)
}

func UnmarshalInvoiceExtractionRequested(data []byte) (InvoiceExtractionRequested, error) {
	var event InvoiceExtractionRequested
	if err := json.Unmarshal(data, &event); err != nil {
		return InvoiceExtractionRequested{}, err
	}

	if err := event.Validate(); err != nil {
		return InvoiceExtractionRequested{}, err
	}

	return event, nil
}

func DecodeInvoiceExtractionRequestedFromCloudWatchEvent(event awsevents.CloudWatchEvent) (InvoiceExtractionRequested, error) {
	if event.DetailType != InvoiceExtractionRequestedDetailType {
		return InvoiceExtractionRequested{}, fmt.Errorf("unexpected detail type: %s", event.DetailType)
	}

	return UnmarshalInvoiceExtractionRequested(event.Detail)
}
