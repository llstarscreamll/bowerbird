package cloudevents

import (
	"encoding/json"
	"time"
)

const SpecVersion = "1.0"

type Event struct {
	ID                string          `json:"id"`
	Source            string          `json:"source"`
	SpecVersion       string          `json:"specversion"`
	Type              string          `json:"type"`
	Time              string          `json:"time"`
	Data              json.RawMessage `json:"data"`
	TenantSlug        string          `json:"tenant_slug,omitempty"`
	CorrelationID     string          `json:"correlation_id,omitempty"`
	TenantAttestation string          `json:"tenant_attestation,omitempty"`
}

func NewEvent(id, source, detailType, tenantSlug, correlationID string, payload []byte) Event {
	return Event{
		ID:            id,
		Source:        source,
		SpecVersion:   SpecVersion,
		Type:          detailType,
		Time:          time.Now().UTC().Format(time.RFC3339Nano),
		Data:          json.RawMessage(payload),
		TenantSlug:    tenantSlug,
		CorrelationID: correlationID,
	}
}

func MarshalEvent(event Event) ([]byte, error) {
	return json.Marshal(event)
}

func UnmarshalEvent(data []byte) (Event, error) {
	var event Event
	err := json.Unmarshal(data, &event)
	return event, err
}
