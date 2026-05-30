package domain

import (
	"fmt"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
)

const (
	InboxMessageReceivedDetailType    = contractevents.InboxMessageReceivedDetailType
	InboxMessageReceivedSource        = contractevents.InboxMessageReceivedSource
	InboxMessageReceivedSchemaVersion = contractevents.InboxMessageReceivedSchemaVersion
)

type AttachmentRef = contractevents.AttachmentRef

type InboxMessageReceived = contractevents.InboxMessageReceived

type NewInboxMessageReceivedInput struct {
	EventID           string
	OccurredAt        string
	TenantSlug        string
	AccountID         string
	Provider          string
	ProviderMessage   *MailMessage
	MessageInternalID string
	AttachmentRefs    []AttachmentRef
}

func NewInboxMessageReceived(input NewInboxMessageReceivedInput) (InboxMessageReceived, error) {
	if input.EventID == "" {
		return InboxMessageReceived{}, fmt.Errorf("event id is required")
	}
	if input.OccurredAt == "" {
		return InboxMessageReceived{}, fmt.Errorf("occurred at is required")
	}
	if input.TenantSlug == "" {
		return InboxMessageReceived{}, fmt.Errorf("tenant slug is required")
	}
	if input.AccountID == "" {
		return InboxMessageReceived{}, fmt.Errorf("account id is required")
	}
	if input.Provider == "" {
		return InboxMessageReceived{}, fmt.Errorf("provider is required")
	}
	if input.ProviderMessage == nil {
		return InboxMessageReceived{}, fmt.Errorf("provider message is required")
	}
	if input.ProviderMessage.ID == "" {
		return InboxMessageReceived{}, fmt.Errorf("provider message id is required")
	}
	if input.MessageInternalID == "" {
		return InboxMessageReceived{}, fmt.Errorf("message internal id is required")
	}

	event := InboxMessageReceived{
		EventID:           input.EventID,
		OccurredAt:        input.OccurredAt,
		TenantSlug:        input.TenantSlug,
		AccountID:         input.AccountID,
		Provider:          input.Provider,
		ProviderMessageID: input.ProviderMessage.ID,
		MessageInternalID: input.MessageInternalID,
		AttachmentRefs:    append([]AttachmentRef(nil), input.AttachmentRefs...),
	}

	if input.ProviderMessage.Subject != "" {
		event.Subject = input.ProviderMessage.Subject
	}
	if input.ProviderMessage.Sender != "" {
		event.Sender = input.ProviderMessage.Sender
	}
	if input.ProviderMessage.ReceivedAt != nil {
		event.ReceivedAt = input.ProviderMessage.ReceivedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return event, nil
}

func MarshalInboxMessageReceived(event InboxMessageReceived) ([]byte, error) {
	return contractevents.MarshalInboxMessageReceived(event)
}

func UnmarshalInboxMessageReceived(data []byte) (InboxMessageReceived, error) {
	return contractevents.UnmarshalInboxMessageReceived(data)
}

func DecodeInboxMessageReceivedFromCloudWatchEvent(event awsevents.CloudWatchEvent) (InboxMessageReceived, error) {
	return contractevents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
}
