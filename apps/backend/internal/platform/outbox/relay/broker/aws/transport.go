package awsbroker

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebTypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/outbox/cloudevents"
	"github.com/bowerbird/internal/platform/outbox/store"
)

type Transport struct {
	eventBridge       eventBridgeClient
	sqs               sqsClient
	eventBus          string
	queueURL          string
	attestationSecret string
}

type eventBridgeClient interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type sqsClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func NewTransport(eventBridge *eventbridge.Client, sqsClient *sqs.Client, eventBus, queueURL, attestationSecret string) *Transport {
	return NewTransportWithClients(eventBridge, sqsClient, eventBus, queueURL, attestationSecret)
}

func NewTransportWithClients(eventBridge eventBridgeClient, sqsClient sqsClient, eventBus, queueURL, attestationSecret string) *Transport {
	return &Transport{
		eventBridge:       eventBridge,
		sqs:               sqsClient,
		eventBus:          eventBus,
		queueURL:          queueURL,
		attestationSecret: attestationSecret,
	}
}

func (t *Transport) DeliverEvent(ctx context.Context, row store.EventRow) error {
	ce := cloudevents.NewEvent(row.ID, row.Source, row.DetailType, row.TenantSlug, row.CorrelationID, row.Payload)
	ce.TenantAttestation = attestation.Sign(t.attestationSecret, row.ID, row.TenantSlug, row.DetailType)
	detail, err := cloudevents.MarshalEvent(ce)
	if err != nil {
		return err
	}

	_, err = t.eventBridge.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebTypes.PutEventsRequestEntry{
			{
				EventBusName: aws.String(t.eventBus),
				Source:       aws.String(row.Source),
				DetailType:   aws.String(row.DetailType),
				Detail:       aws.String(string(detail)),
			},
		},
	})
	return err
}

func (t *Transport) DeliverJob(ctx context.Context, row store.JobRow) error {
	tenantAttestation := attestation.Sign(t.attestationSecret, row.ID, row.TenantSlug, row.JobType)
	envelope := map[string]any{
		"job_type":           row.JobType,
		"tenant_slug":        row.TenantSlug,
		"correlation_id":     row.CorrelationID,
		"message_id":         row.ID,
		"payload":            json.RawMessage(row.Payload),
		"tenant_attestation": tenantAttestation,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	attrs := map[string]sqsTypes.MessageAttributeValue{
		"JobType": {
			DataType:    aws.String("String"),
			StringValue: aws.String(row.JobType),
		},
		"TenantID": {
			DataType:    aws.String("String"),
			StringValue: aws.String(row.TenantSlug),
		},
		"TenantAttestation": {
			DataType:    aws.String("String"),
			StringValue: aws.String(tenantAttestation),
		},
	}
	if row.CorrelationID != "" {
		attrs["CorrelationID"] = sqsTypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(row.CorrelationID),
		}
	}

	_, err = t.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(t.queueURL),
		MessageBody:       aws.String(string(body)),
		MessageAttributes: attrs,
	})
	return err
}
