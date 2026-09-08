package jobs

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/tenant"
)

type Router struct {
	handlers map[string]JobHandler
	verifier *attestation.Verifier
}

func NewRouter(verifier *attestation.Verifier, handlers ...JobHandler) Router {
	if verifier == nil {
		panic("attestation verifier is required")
	}
	routes := make(map[string]JobHandler)
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		routes[handler.JobType()] = handler
	}
	return Router{handlers: routes, verifier: verifier}
}

func (r Router) JobTypes() []string {
	keys := make([]string, 0, len(r.handlers))
	for jobType := range r.handlers {
		keys = append(keys, jobType)
	}
	sort.Strings(keys)
	return keys
}

func (r Router) HandleJob(ctx context.Context, msg JobMessage) error {
	handler, found := r.handlers[msg.JobType]
	if found && handler.Scope() == ScopePlatform {
		return r.handlePlatformJob(ctx, handler, msg)
	}
	return r.handleTenantJob(ctx, handler, found, msg)
}

func (r Router) handlePlatformJob(ctx context.Context, handler JobHandler, msg JobMessage) error {
	if strings.TrimSpace(msg.TenantSlug) != "" {
		return ErrJobScopeMismatch
	}
	if err := r.verifier.Verify(msg.MessageID, attestation.PlatformSubject, msg.JobType, msg.TenantAttestation); err != nil {
		return err
	}
	if err := handler.Handle(ctx, msg); err != nil {
		return err
	}
	log.Printf("job routed: id=%s type=%s tenant=%s", msg.MessageID, msg.JobType, attestation.PlatformSubject)
	return nil
}

func (r Router) handleTenantJob(ctx context.Context, handler JobHandler, found bool, msg JobMessage) error {
	slug := strings.TrimSpace(msg.TenantSlug)
	if slug == "" {
		return tenant.ErrNoTenantIdInContext
	}
	if slug == attestation.PlatformSubject {
		return ErrJobScopeMismatch
	}
	if err := r.verifier.Verify(msg.MessageID, slug, msg.JobType, msg.TenantAttestation); err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, slug)
	if found {
		if err := handler.Handle(msgCtx, msg); err != nil {
			return err
		}
		log.Printf("job routed: id=%s type=%s tenant=%s", msg.MessageID, msg.JobType, slug)
		return nil
	}

	log.Printf("job processed without handler: id=%s type=%s tenant=%s", msg.MessageID, msg.JobType, slug)
	return nil
}

func (r Router) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		if err := r.HandleJob(ctx, jobMessageFromSQS(record)); err != nil {
			return err
		}
	}
	return nil
}

func jobMessageFromSQS(record events.SQSMessage) JobMessage {
	msg := JobMessage{MessageID: record.MessageId, Body: []byte(record.Body)}
	if attr, ok := record.MessageAttributes["MessageID"]; ok && attr.StringValue != nil {
		msg.MessageID = *attr.StringValue
	}
	if attr, ok := record.MessageAttributes["TenantID"]; ok && attr.StringValue != nil {
		msg.TenantSlug = *attr.StringValue
	}
	if attr, ok := record.MessageAttributes["JobType"]; ok && attr.StringValue != nil {
		msg.JobType = *attr.StringValue
	}
	if attr, ok := record.MessageAttributes["CorrelationID"]; ok && attr.StringValue != nil {
		msg.CorrelationID = *attr.StringValue
	}
	if attr, ok := record.MessageAttributes["TenantAttestation"]; ok && attr.StringValue != nil {
		msg.TenantAttestation = *attr.StringValue
	}
	applyJobEnvelope(&msg)
	return msg
}

func applyJobEnvelope(msg *JobMessage) {
	var envelope struct {
		MessageID         string `json:"message_id"`
		JobType           string `json:"job_type"`
		TenantSlug        string `json:"tenant_slug"`
		CorrelationID     string `json:"correlation_id"`
		TenantAttestation string `json:"tenant_attestation"`
	}
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		return
	}
	if envelope.MessageID != "" {
		msg.MessageID = envelope.MessageID
	}
	if envelope.JobType != "" {
		msg.JobType = envelope.JobType
	}
	if envelope.TenantSlug != "" {
		msg.TenantSlug = envelope.TenantSlug
	}
	if envelope.CorrelationID != "" {
		msg.CorrelationID = envelope.CorrelationID
	}
	if envelope.TenantAttestation != "" {
		msg.TenantAttestation = envelope.TenantAttestation
	}
}

func NewHandler(verifier *attestation.Verifier, handlers ...JobHandler) Router {
	return NewRouter(verifier, handlers...)
}

type Handler = Router
