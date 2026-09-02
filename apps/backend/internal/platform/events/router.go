package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/outbox/cloudevents"
	"github.com/bowerbird/internal/platform/tenant"
)

type Router struct {
	handlers map[string]IntegrationEventHandler
	verifier *attestation.Verifier
}

func NewRouter(verifier *attestation.Verifier, subscribers ...IntegrationEventHandler) Router {
	routes := make(map[string]IntegrationEventHandler)
	for _, subscriber := range subscribers {
		if subscriber == nil {
			continue
		}
		routes[subscriber.DetailType()] = subscriber
	}
	return Router{handlers: routes, verifier: verifier}
}

func (r Router) HandleIntegrationEvent(ctx context.Context, event IntegrationEvent) error {
	if r.verifier != nil {
		if err := r.verifier.Verify(event.ID, event.TenantSlug, event.DetailType, event.TenantAttestation); err != nil {
			return err
		}
	}

	msgCtx := ctx
	if event.TenantSlug != "" {
		msgCtx = tenant.WithTenantID(msgCtx, event.TenantSlug)
	}

	if handler, ok := r.handlers[event.DetailType]; ok {
		if err := handler.Handle(msgCtx, event); err != nil {
			return err
		}
		log.Printf("integration event routed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
		return nil
	}

	log.Printf("integration event processed: id=%s type=%s source=%s", event.ID, event.DetailType, event.Source)
	return nil
}

func (r Router) HandleEventBridgeEvent(ctx context.Context, event events.CloudWatchEvent) error {
	integrationEvent := IntegrationEvent{
		ID:         event.ID,
		Source:     event.Source,
		DetailType: event.DetailType,
		Detail:     event.Detail,
	}

	var ce cloudevents.Event
	if err := json.Unmarshal(event.Detail, &ce); err == nil && ce.SpecVersion != "" {
		integrationEvent.ID = ce.ID
		integrationEvent.Source = ce.Source
		integrationEvent.DetailType = ce.Type
		integrationEvent.Detail = ce.Data
		integrationEvent.TenantSlug = ce.TenantSlug
		integrationEvent.CorrelationID = ce.CorrelationID
		integrationEvent.TenantAttestation = ce.TenantAttestation
	}

	return r.HandleIntegrationEvent(ctx, integrationEvent)
}

func (r Router) HandleCloudEventJSON(ctx context.Context, body []byte) error {
	ce, err := cloudevents.UnmarshalEvent(body)
	if err != nil {
		return err
	}
	return r.HandleIntegrationEvent(ctx, IntegrationEvent{
		ID:                ce.ID,
		Source:            ce.Source,
		DetailType:        ce.Type,
		Detail:            ce.Data,
		TenantSlug:        ce.TenantSlug,
		CorrelationID:     ce.CorrelationID,
		TenantAttestation: ce.TenantAttestation,
	})
}

// NewEventHandler is a compatibility alias for NewRouter.
func NewEventHandler(verifier *attestation.Verifier, subscribers ...IntegrationEventHandler) Router {
	return NewRouter(verifier, subscribers...)
}

// EventHandler is a compatibility alias for Router.
type EventHandler = Router
