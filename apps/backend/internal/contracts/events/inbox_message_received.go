package events

import (
	"encoding/json"
	"errors"
	"fmt"

	awsevents "github.com/aws/aws-lambda-go/events"
)

const (
	InboxMessageReceivedDetailType    = "InboxMessageReceived"
	InboxMessageReceivedSource        = "bowerbird.inbox"
	InboxMessageReceivedSchemaVersion = "1.0"
)

type AttachmentRef struct {
	S3Key    string `json:"s3_key"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
}

type InboxMessageReceived struct {
	EventID           string          `json:"event_id"`
	OccurredAt        string          `json:"occurred_at"`
	TenantSlug        string          `json:"tenant_slug"`
	AccountID         string          `json:"account_id"`
	Provider          string          `json:"provider"`
	ProviderMessageID string          `json:"provider_message_id"`
	MessageInternalID string          `json:"message_internal_id"`
	Subject           string          `json:"subject,omitempty"`
	Sender            string          `json:"sender,omitempty"`
	ReceivedAt        string          `json:"received_at,omitempty"`
	AttachmentRefs    []AttachmentRef `json:"attachment_refs,omitempty"`
	RawDataRef        string          `json:"raw_data_ref,omitempty"`
}

func (e InboxMessageReceived) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}

	if e.TenantSlug == "" {
		return errors.New("tenant_slug is required")
	}

	if e.AccountID == "" {
		return errors.New("account_id is required")
	}

	if e.Provider == "" {
		return errors.New("provider is required")
	}

	if e.ProviderMessageID == "" {
		return errors.New("provider_message_id is required")
	}

	if e.MessageInternalID == "" {
		return errors.New("message_internal_id is required")
	}

	return nil
}

func MarshalInboxMessageReceived(event InboxMessageReceived) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func UnmarshalInboxMessageReceived(data []byte) (InboxMessageReceived, error) {
	var event InboxMessageReceived
	if err := json.Unmarshal(data, &event); err != nil {
		return InboxMessageReceived{}, err
	}

	if err := event.Validate(); err != nil {
		return InboxMessageReceived{}, err
	}

	return event, nil
}

func DecodeInboxMessageReceivedFromCloudWatchEvent(event awsevents.CloudWatchEvent) (InboxMessageReceived, error) {
	if event.DetailType != InboxMessageReceivedDetailType {
		return InboxMessageReceived{}, fmt.Errorf("unexpected detail type: %s", event.DetailType)
	}

	return UnmarshalInboxMessageReceived(event.Detail)
}
