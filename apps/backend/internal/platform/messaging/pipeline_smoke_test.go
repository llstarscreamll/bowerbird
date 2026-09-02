package messaging_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	platformEvents "github.com/bowerbird/internal/platform/events"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/outbox/cloudevents"
	"github.com/bowerbird/internal/platform/outbox/relay"
	awsbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/aws"
	rabbitmq "github.com/bowerbird/internal/platform/outbox/relay/broker/rabbitmq"
	"github.com/bowerbird/internal/platform/outbox/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type smokeStore struct {
	events []store.EventRow
	jobs   []store.JobRow
	mu     sync.Mutex
	status map[string]string
}

func newSmokeStore(events []store.EventRow, jobs []store.JobRow) *smokeStore {
	return &smokeStore{events: events, jobs: jobs, status: map[string]string{}}
}

func (m *smokeStore) InsertEvent(context.Context, pgx.Tx, store.InsertEventInput) error   { return nil }
func (m *smokeStore) InsertJob(context.Context, pgx.Tx, store.InsertJobInput) error       { return nil }
func (m *smokeStore) InsertEventStandalone(context.Context, store.InsertEventInput) error { return nil }
func (m *smokeStore) InsertJobStandalone(context.Context, store.InsertJobInput) error     { return nil }

func (m *smokeStore) ClaimPendingEvents(_ context.Context, limit int) ([]store.EventRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || len(m.events) == 0 {
		return nil, nil
	}
	if limit > len(m.events) {
		limit = len(m.events)
	}
	out := append([]store.EventRow(nil), m.events[:limit]...)
	m.events = m.events[limit:]
	return out, nil
}

func (m *smokeStore) ClaimPendingJobs(_ context.Context, limit int) ([]store.JobRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || len(m.jobs) == 0 {
		return nil, nil
	}
	if limit > len(m.jobs) {
		limit = len(m.jobs)
	}
	out := append([]store.JobRow(nil), m.jobs[:limit]...)
	m.jobs = m.jobs[limit:]
	return out, nil
}

func (m *smokeStore) MarkEventProcessed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[id] = store.StatusProcessed
	return nil
}

func (m *smokeStore) MarkJobProcessed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[id] = store.StatusProcessed
	return nil
}

func (m *smokeStore) MarkEventFailed(context.Context, string, string, int, int) error { return nil }
func (m *smokeStore) MarkJobFailed(context.Context, string, string, int, int) error   { return nil }
func (m *smokeStore) IncrementEventAttempt(context.Context, string, string) error     { return nil }
func (m *smokeStore) IncrementJobAttempt(context.Context, string, string) error       { return nil }
func (m *smokeStore) CountPending(context.Context) (int64, int64, error)              { return 0, 0, nil }
func (m *smokeStore) PurgeTerminal(context.Context, time.Time) (int64, int64, error) {
	return 0, 0, nil
}

type captureEventBridge struct {
	last *eventbridge.PutEventsInput
}

func (f *captureEventBridge) PutEvents(_ context.Context, in *eventbridge.PutEventsInput, _ ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	f.last = in
	return &eventbridge.PutEventsOutput{}, nil
}

type captureSQS struct {
	last *sqs.SendMessageInput
}

func (f *captureSQS) SendMessage(_ context.Context, in *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.last = in
	return &sqs.SendMessageOutput{}, nil
}

type smokeEventHandler struct {
	handled string
}

func (h *smokeEventHandler) DetailType() string { return "SmokeEvent" }
func (h *smokeEventHandler) Handle(_ context.Context, event platformEvents.IntegrationEvent) error {
	h.handled = event.ID
	return nil
}

type smokeJobHandler struct {
	handled string
}

func (h *smokeJobHandler) JobType() string { return "SmokeJob" }
func (h *smokeJobHandler) Handle(_ context.Context, msg platformJobs.JobMessage) error {
	h.handled = msg.MessageID
	return nil
}

const smokeAttestationSecret = "smoke-attestation-secret"

func TestPipelineAWSSmoke(t *testing.T) {
	st := newSmokeStore(
		[]store.EventRow{{
			ID: "evt-smoke-1", Source: "bowerbird.test", DetailType: "SmokeEvent",
			TenantSlug: "acme", CorrelationID: "corr-1", Payload: []byte(`{"ok":true}`), MaxAttempts: 3,
		}},
		[]store.JobRow{{
			ID: "job-smoke-1", JobType: "SmokeJob", TenantSlug: "acme",
			CorrelationID: "corr-2", Payload: []byte(`{"task":1}`), MaxAttempts: 3,
		}},
	)

	eb := &captureEventBridge{}
	sqsClient := &captureSQS{}
	transport := awsbroker.NewTransportWithClients(eb, sqsClient, "bus", "https://sqs.example/queue", smokeAttestationSecret)

	r := relay.New(st, transport, relay.Config{BatchSize: 10})
	require.NoError(t, r.RunOnce(context.Background()))
	require.Equal(t, store.StatusProcessed, st.status["evt-smoke-1"])
	require.Equal(t, store.StatusProcessed, st.status["job-smoke-1"])
	require.NotNil(t, eb.last)
	require.NotNil(t, sqsClient.last)

	eventHandler := &smokeEventHandler{}
	verifier := attestation.NewVerifier(smokeAttestationSecret)
	eventsRouter := platformEvents.NewRouter(verifier, eventHandler)
	require.NoError(t, eventsRouter.HandleEventBridgeEvent(context.Background(), events.CloudWatchEvent{
		ID:         "evt-smoke-1",
		Source:     "bowerbird.test",
		DetailType: "SmokeEvent",
		Detail:     json.RawMessage(*eb.last.Entries[0].Detail),
	}))
	require.Equal(t, "evt-smoke-1", eventHandler.handled)

	jobHandler := &smokeJobHandler{}
	jobsRouter := platformJobs.NewRouter(verifier, jobHandler)
	jobType := aws.ToString(sqsClient.last.MessageAttributes["JobType"].StringValue)
	tenantID := aws.ToString(sqsClient.last.MessageAttributes["TenantID"].StringValue)
	correlationID := aws.ToString(sqsClient.last.MessageAttributes["CorrelationID"].StringValue)
	tenantAttestation := aws.ToString(sqsClient.last.MessageAttributes["TenantAttestation"].StringValue)
	require.NoError(t, jobsRouter.HandleSQSEvent(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{
		MessageId: "job-smoke-1",
		Body:      aws.ToString(sqsClient.last.MessageBody),
		MessageAttributes: map[string]events.SQSMessageAttribute{
			"JobType":           {StringValue: &jobType},
			"TenantID":          {StringValue: &tenantID},
			"CorrelationID":     {StringValue: &correlationID},
			"TenantAttestation": {StringValue: &tenantAttestation},
		},
	}}}))
	require.Equal(t, "job-smoke-1", jobHandler.handled)
}

func TestPipelineOnPremSmoke(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if dsn == "" || rabbitURL == "" {
		t.Skip("TEST_DATABASE_URL and RABBITMQ_URL required for on-prem pipeline smoke")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool := connectPool(t, ctx, dsn)
	defer pool.Close()

	eventID := "smoke-onprem-evt-" + time.Now().Format("150405")
	jobID := "smoke-onprem-job-" + time.Now().Format("150405")
	cleanupSmokeRows(t, ctx, pool, eventID, jobID)

	st := store.NewPostgresStore(pool)
	require.NoError(t, st.InsertEventStandalone(ctx, store.InsertEventInput{
		ID: eventID, TenantSlug: "acme", Source: "bowerbird.test", DetailType: "SmokeEvent",
		Payload: []byte(`{"smoke":true}`), CorrelationID: "corr-onprem", MaxAttempts: 3,
	}))
	require.NoError(t, st.InsertJobStandalone(ctx, store.InsertJobInput{
		ID: jobID, TenantSlug: "acme", JobType: "SmokeJob",
		Payload: []byte(`{"smoke":true}`), CorrelationID: "corr-onprem", MaxAttempts: 3,
	}))

	conn := rabbitmq.NewConnection(rabbitURL)
	require.NoError(t, conn.Connect())
	ch, err := conn.Channel()
	require.NoError(t, err)
	require.NoError(t, rabbitmq.DeclareTopology(ch))
	require.NoError(t, rabbitmq.BindJobsQueue(ch, "SmokeJob"))
	require.NoError(t, ch.Close())

	transport, err := rabbitmq.NewTransport(conn, smokeAttestationSecret, "SmokeJob")
	require.NoError(t, err)

	r := relay.New(st, transport, relay.Config{BatchSize: 10})
	require.NoError(t, r.RunOnce(ctx))

	require.Equal(t, store.StatusProcessed, rowStatus(t, ctx, pool, "outbox_events", eventID))
	require.Equal(t, store.StatusProcessed, rowStatus(t, ctx, pool, "outbox_jobs", jobID))

	eventBody := consumeOne(t, ctx, rabbitURL, rabbitmq.EventsQueue)
	ce, err := cloudevents.UnmarshalEvent(eventBody)
	require.NoError(t, err)
	require.Equal(t, eventID, ce.ID)

	jobBody := consumeOne(t, ctx, rabbitURL, rabbitmq.JobsQueue)
	require.Contains(t, string(jobBody), jobID)

	eventHandler := &smokeEventHandler{}
	onPremVerifier := attestation.NewVerifier(smokeAttestationSecret)
	require.NoError(t, platformEvents.NewRouter(onPremVerifier, eventHandler).HandleCloudEventJSON(ctx, eventBody))
	require.Equal(t, eventID, eventHandler.handled)

	jobHandler := &smokeJobHandler{}
	var jobEnvelope map[string]string
	require.NoError(t, json.Unmarshal(jobBody, &jobEnvelope))
	require.NoError(t, platformJobs.NewRouter(onPremVerifier, jobHandler).HandleJob(ctx, platformJobs.JobMessage{
		MessageID: jobID, JobType: "SmokeJob", TenantSlug: "acme", Body: jobBody,
		TenantAttestation: jobEnvelope["tenant_attestation"],
	}))
	require.Equal(t, jobID, jobHandler.handled)
}

func connectPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	return pool
}

func cleanupSmokeRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventID, jobID string) {
	t.Helper()
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE id = $1`, eventID)
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_jobs WHERE id = $1`, jobID)
}

func rowStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, id string) string {
	t.Helper()
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM `+table+` WHERE id = $1`, id).Scan(&status)
	require.NoError(t, err)
	return status
}

func consumeOne(t *testing.T, ctx context.Context, rabbitURL, queue string) []byte {
	t.Helper()
	conn, err := amqp.Dial(rabbitURL)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Second)
	}
	for time.Now().Before(deadline) {
		msg, ok, err := ch.Get(queue, true)
		if err != nil {
			require.NoError(t, err)
		}
		if ok {
			return msg.Body
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("no message received from queue " + queue)
	return nil
}
