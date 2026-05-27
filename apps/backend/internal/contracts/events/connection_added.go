package events

import (
	"encoding/json"
	"errors"
)

const (
	ConnectionAddedDetailType    = "ConnectionAdded"
	ConnectionAddedSource        = "bowerbird.connections"
	ConnectionAddedSchemaVersion = "1.0"
)

type ConnectionAdded struct {
	EventID              string `json:"event_id"`
	OccurredAt           string `json:"occurred_at"`
	TenantSlug           string `json:"tenant_slug"`
	ConnectionID         string `json:"connection_id"`
	Provider             string `json:"provider"`
	ProviderAccountEmail string `json:"provider_account_email"`
}

func (e ConnectionAdded) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if e.TenantSlug == "" {
		return errors.New("tenant_slug is required")
	}
	if e.ConnectionID == "" {
		return errors.New("connection_id is required")
	}
	if e.Provider == "" {
		return errors.New("provider is required")
	}
	return nil
}

func MarshalConnectionAdded(event ConnectionAdded) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(event)
}

func UnmarshalConnectionAdded(data []byte) (ConnectionAdded, error) {
	var event ConnectionAdded
	if err := json.Unmarshal(data, &event); err != nil {
		return ConnectionAdded{}, err
	}
	if err := event.Validate(); err != nil {
		return ConnectionAdded{}, err
	}
	return event, nil
}
