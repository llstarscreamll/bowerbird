package jobs

import (
	"context"
	"log"
	"sort"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/tenant"
)

type Router struct {
	handlers map[string]JobHandler
	verifier *attestation.Verifier
}

func NewRouter(verifier *attestation.Verifier, handlers ...JobHandler) Router {
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
	if r.verifier != nil {
		if err := r.verifier.Verify(msg.MessageID, msg.TenantSlug, msg.JobType, msg.TenantAttestation); err != nil {
			return err
		}
	}

	msgCtx := ctx
	if msg.TenantSlug != "" {
		msgCtx = tenant.WithTenantID(msgCtx, msg.TenantSlug)
	}

	if handler, found := r.handlers[msg.JobType]; found {
		if err := handler.Handle(msgCtx, msg); err != nil {
			return err
		}
		log.Printf("job routed: id=%s type=%s tenant=%s", msg.MessageID, msg.JobType, msg.TenantSlug)
		return nil
	}

	log.Printf("job processed without handler: id=%s type=%s tenant=%s", msg.MessageID, msg.JobType, msg.TenantSlug)
	return nil
}

func (r Router) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		msg := JobMessage{MessageID: record.MessageId, Body: []byte(record.Body)}
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
		if err := r.HandleJob(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func NewHandler(verifier *attestation.Verifier, handlers ...JobHandler) Router {
	return NewRouter(verifier, handlers...)
}

type Handler = Router
