package awsbroker_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/bowerbird/internal/platform/outbox/cloudevents"
	awsbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/aws"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/stretchr/testify/require"
)

type fakeEventBridge struct {
	last *eventbridge.PutEventsInput
	err  error
}

func (f *fakeEventBridge) PutEvents(_ context.Context, in *eventbridge.PutEventsInput, _ ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	f.last = in
	if f.err != nil {
		return nil, f.err
	}
	return &eventbridge.PutEventsOutput{}, nil
}

type fakeSQS struct {
	last *sqs.SendMessageInput
}

func (f *fakeSQS) SendMessage(_ context.Context, in *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.last = in
	return &sqs.SendMessageOutput{}, nil
}

func TestTransportDeliverEventUsesCloudEvents(t *testing.T) {
	eb := &fakeEventBridge{}
	sqsClient := &fakeSQS{}
	tr := awsbroker.NewTransportWithClients(eb, sqsClient, "bus", "queue-url", "test-attestation-secret")

	row := store.EventRow{
		ID: "evt-1", Source: "bowerbird.inbox", DetailType: "InboxMessageReceived",
		TenantSlug: "acme", CorrelationID: "corr-1", Payload: []byte(`{"k":"v"}`),
	}
	require.NoError(t, tr.DeliverEvent(context.Background(), row))
	require.Len(t, eb.last.Entries, 1)

	var ce cloudevents.Event
	require.NoError(t, json.Unmarshal([]byte(*eb.last.Entries[0].Detail), &ce))
	require.Equal(t, cloudevents.SpecVersion, ce.SpecVersion)
	require.Equal(t, "acme", ce.TenantSlug)
	require.Equal(t, "corr-1", ce.CorrelationID)
	require.NotEmpty(t, ce.TenantAttestation)
}

func TestTransportDeliverJobSetsAttributes(t *testing.T) {
	eb := &fakeEventBridge{}
	sqsClient := &fakeSQS{}
	tr := awsbroker.NewTransportWithClients(eb, sqsClient, "bus", "https://sqs/queue", "test-attestation-secret")

	row := store.JobRow{
		ID: "job-1", JobType: "InvoiceExtractionRequested", TenantSlug: "acme",
		CorrelationID: "corr-2", Payload: []byte(`{"id":"x"}`),
	}
	require.NoError(t, tr.DeliverJob(context.Background(), row))
	require.NotNil(t, sqsClient.last)
	require.Equal(t, aws.String("InvoiceExtractionRequested"), sqsClient.last.MessageAttributes["JobType"].StringValue)
	require.Equal(t, aws.String("acme"), sqsClient.last.MessageAttributes["TenantID"].StringValue)
	require.Equal(t, aws.String("corr-2"), sqsClient.last.MessageAttributes["CorrelationID"].StringValue)
	require.NotEmpty(t, aws.ToString(sqsClient.last.MessageAttributes["TenantAttestation"].StringValue))
}
