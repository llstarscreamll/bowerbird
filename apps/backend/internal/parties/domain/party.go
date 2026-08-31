package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingPartyName = errors.New("missing party name")
	ErrMissingTaxID     = errors.New("missing party tax id")
	ErrPartyIDRequired  = errors.New("party id is required")
)

const (
	RoleSupplier = "supplier"
	RoleCustomer = "customer"

	StatusProvisional = "provisional"
	StatusConfirmed   = "confirmed"
)

// Party is the trading-partner aggregate root (supplier/customer identity by tax id).
type Party struct {
	ID        string
	TaxID     string
	Name      string
	Roles     []string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewProvisionalSupplier bootstraps a provisional supplier from invoice issuer evidence.
// Caller must supply a parsed TaxID.
func NewProvisionalSupplier(id string, taxID TaxID, name string, now time.Time) Party {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = taxID.String()
	}
	now = now.UTC()
	return Party{
		ID:        id,
		TaxID:     taxID.String(),
		Name:      displayName,
		Roles:     []string{RoleSupplier},
		Status:    StatusProvisional,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewConfirmedParty creates a user-registered confirmed party.
func NewConfirmedParty(id string, taxID TaxID, name string, roles PartyRoles, now time.Time) (Party, error) {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		return Party{}, ErrMissingPartyName
	}
	if strings.TrimSpace(id) == "" {
		return Party{}, ErrPartyIDRequired
	}
	now = now.UTC()
	return Party{
		ID:        id,
		TaxID:     taxID.String(),
		Name:      displayName,
		Roles:     roles.Strings(),
		Status:    StatusConfirmed,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p Party) HasRole(role string) bool {
	for _, existing := range p.Roles {
		if existing == role {
			return true
		}
	}
	return false
}

// EnsureSupplierRole adds the supplier role from invoice issuer evidence.
func (p *Party) EnsureSupplierRole(now time.Time) bool {
	if p.HasRole(RoleSupplier) {
		return false
	}
	p.Roles = append(p.Roles, RoleSupplier)
	p.UpdatedAt = now.UTC()
	return true
}

// Rename updates the display name with validation.
func (p *Party) Rename(name string, now time.Time) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrMissingPartyName
	}
	if p.Name == trimmed {
		return nil
	}
	p.Name = trimmed
	p.UpdatedAt = now.UTC()
	return nil
}

// AssignRoles replaces roles with a validated set.
func (p *Party) AssignRoles(roles PartyRoles, now time.Time) {
	if partyRolesEqual(p.Roles, roles) {
		return
	}
	p.Roles = roles.Strings()
	p.UpdatedAt = now.UTC()
}

func partyRolesEqual(stored []string, desired PartyRoles) bool {
	current, err := ParsePartyRoles(stored)
	if err != nil {
		return false
	}
	return current.Equals(desired)
}

// UpdateProfile applies manual profile changes (name and/or roles).
func (p *Party) UpdateProfile(name *string, roles *PartyRoles, now time.Time) (changed bool, err error) {
	if name == nil && roles == nil {
		return false, nil
	}
	before := p.UpdatedAt
	if name != nil {
		if err := p.Rename(*name, now); err != nil {
			return false, err
		}
	}
	if roles != nil {
		p.AssignRoles(*roles, now)
	}
	return !p.UpdatedAt.Equal(before), nil
}

func NormalizeTaxID(taxID string) string {
	return strings.TrimSpace(taxID)
}
