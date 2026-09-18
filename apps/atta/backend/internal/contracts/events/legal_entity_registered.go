package events

import (
	"encoding/json"
	"errors"
)

const (
	LegalEntityRegisteredSchemaVersion = "1.0"
	LegalEntityRegisteredSource        = "atta.legalentities"
	LegalEntityRegisteredDetailType    = "LegalEntityRegistered"
)

type LegalEntityRegistered struct {
	EventID    string `json:"event_id"`
	OccurredAt string `json:"occurred_at"`
	TenantSlug string `json:"tenant_slug"`
}

func (e LegalEntityRegistered) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if e.TenantSlug == "" {
		return errors.New("tenant_slug is required")
	}
	return nil
}

func MarshalLegalEntityRegistered(event LegalEntityRegistered) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(event)
}

func UnmarshalLegalEntityRegistered(data []byte) (LegalEntityRegistered, error) {
	var event LegalEntityRegistered
	if err := json.Unmarshal(data, &event); err != nil {
		return LegalEntityRegistered{}, err
	}
	if err := event.Validate(); err != nil {
		return LegalEntityRegistered{}, err
	}
	return event, nil
}
