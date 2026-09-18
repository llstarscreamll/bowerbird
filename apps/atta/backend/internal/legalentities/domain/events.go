package domain

import "time"

const (
	EventRegistered      = "LegalEntityRegistered"
	EventIdentityChanged = "LegalEntityIdentityChanged"
)

// Event is a domain event raised by LegalEntity.
type Event struct {
	Name       string
	OccurredAt time.Time
}

func RequiresRegistrationNotice(events []Event) bool {
	for _, ev := range events {
		if ev.Name == EventRegistered || ev.Name == EventIdentityChanged {
			return true
		}
	}
	return false
}
