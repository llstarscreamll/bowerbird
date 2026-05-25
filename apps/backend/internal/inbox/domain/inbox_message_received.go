package domain

import (
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

func MarshalInboxMessageReceived(event InboxMessageReceived) ([]byte, error) {
	return contractevents.MarshalInboxMessageReceived(event)
}

func UnmarshalInboxMessageReceived(data []byte) (InboxMessageReceived, error) {
	return contractevents.UnmarshalInboxMessageReceived(data)
}

func DecodeInboxMessageReceivedFromCloudWatchEvent(event awsevents.CloudWatchEvent) (InboxMessageReceived, error) {
	return contractevents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
}
