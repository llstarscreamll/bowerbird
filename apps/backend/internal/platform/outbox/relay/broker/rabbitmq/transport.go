package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/outbox/cloudevents"
	"github.com/bowerbird/internal/platform/outbox/store"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Transport struct {
	conn              *Connection
	ch                *amqp.Channel
	attestationSecret string
}

func NewTransport(conn *Connection, attestationSecret string, jobRoutingKeys ...string) (*Transport, error) {
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := DeclareTopology(ch); err != nil {
		return nil, err
	}
	if len(jobRoutingKeys) > 0 {
		if err := BindJobsQueue(ch, jobRoutingKeys...); err != nil {
			return nil, err
		}
	}
	if err := ch.Confirm(false); err != nil {
		return nil, err
	}
	return &Transport{conn: conn, ch: ch, attestationSecret: attestationSecret}, nil
}

func (t *Transport) DeliverEvent(ctx context.Context, row store.EventRow) error {
	ce := cloudevents.NewEvent(row.ID, row.Source, row.DetailType, row.TenantSlug, row.CorrelationID, row.Payload)
	ce.TenantAttestation = attestation.Sign(t.attestationSecret, row.ID, row.TenantSlug, row.DetailType)
	body, err := cloudevents.MarshalEvent(ce)
	if err != nil {
		return err
	}

	headers := amqp.Table{
		"tenant_slug":         row.TenantSlug,
		"correlation_id":      row.CorrelationID,
		attestation.HeaderKey: ce.TenantAttestation,
	}
	return t.ch.PublishWithContext(ctx, EventsExchange, row.DetailType, false, false, amqp.Publishing{
		ContentType:  "application/cloudevents+json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      headers,
		MessageId:    row.ID,
	})
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

	headers := amqp.Table{
		"tenant_slug":         row.TenantSlug,
		"correlation_id":      row.CorrelationID,
		"job_type":            row.JobType,
		attestation.HeaderKey: tenantAttestation,
	}
	return t.ch.PublishWithContext(ctx, JobsExchange, row.JobType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      headers,
		MessageId:    row.ID,
	})
}
