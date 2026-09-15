package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingTaxID      = errors.New("missing tax id")
	ErrMissingLegalName  = errors.New("missing legal name")
	ErrInvalidScheme     = errors.New("invalid identification scheme")
	ErrMissingID         = errors.New("legal entity id is required")
	ErrAlreadyRegistered = errors.New("a legal entity is already registered")
)

// LegalEntity is the tenant's own legal identity (not a Party).
type LegalEntity struct {
	ID        string
	TaxID     string
	SchemeID  string
	LegalName string
	CreatedAt time.Time
	UpdatedAt time.Time
	events    []Event
}

func AssertRegistrable(existingCount int) error {
	if existingCount > 0 {
		return ErrAlreadyRegistered
	}
	return nil
}

func NewLegalEntity(id string, taxID TaxID, scheme IdentificationScheme, legalName string, now time.Time) (LegalEntity, error) {
	if strings.TrimSpace(id) == "" {
		return LegalEntity{}, ErrMissingID
	}
	name := strings.TrimSpace(legalName)
	if name == "" {
		return LegalEntity{}, ErrMissingLegalName
	}
	now = now.UTC()
	entity := LegalEntity{
		ID:        id,
		TaxID:     taxID.String(),
		SchemeID:  scheme.String(),
		LegalName: name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	entity.record(Event{Name: EventRegistered, OccurredAt: now})
	return entity, nil
}

func (e *LegalEntity) Rename(name string, now time.Time) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrMissingLegalName
	}
	if e.LegalName == trimmed {
		return nil
	}
	e.LegalName = trimmed
	e.UpdatedAt = now.UTC()
	return nil
}

func (e *LegalEntity) ChangeIdentity(taxID TaxID, scheme IdentificationScheme, now time.Time) bool {
	if e.TaxID == taxID.String() && e.SchemeID == scheme.String() {
		return false
	}
	now = now.UTC()
	e.TaxID = taxID.String()
	e.SchemeID = scheme.String()
	e.UpdatedAt = now
	e.record(Event{Name: EventIdentityChanged, OccurredAt: now})
	return true
}

func (e LegalEntity) ReceivesAs(candidate string) bool {
	stored, err := ParseTaxID(e.TaxID)
	if err != nil {
		return false
	}
	return stored.Matches(candidate)
}

func (e *LegalEntity) PullEvents() []Event {
	out := e.events
	e.events = nil
	return out
}

func (e *LegalEntity) record(ev Event) {
	e.events = append(e.events, ev)
}
