package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingPartyName = errors.New("missing party name")
	ErrMissingTaxID     = errors.New("missing party tax id")
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
func NewProvisionalSupplier(id, taxID, name string, now time.Time) (Party, error) {
	normalizedTaxID := NormalizeTaxID(taxID)
	if normalizedTaxID == "" {
		return Party{}, ErrMissingTaxID
	}
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = normalizedTaxID
	}
	now = now.UTC()
	return Party{
		ID:        id,
		TaxID:     normalizedTaxID,
		Name:      displayName,
		Roles:     []string{RoleSupplier},
		Status:    StatusProvisional,
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

// EnsureRole adds a role if missing. Prefer EnsureSupplierRole for issuer resolution.
func (p *Party) EnsureRole(role string) {
	if p.HasRole(role) {
		return
	}
	p.Roles = append(p.Roles, role)
}

// EnsureSupplierRole adds the supplier role and bumps UpdatedAt when changed.
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
	p.Name = trimmed
	p.UpdatedAt = now.UTC()
	return nil
}

// ReplaceRoles replaces the role set (caller supplies the desired list).
func (p *Party) ReplaceRoles(roles []string, now time.Time) {
	p.Roles = append([]string{}, roles...)
	p.UpdatedAt = now.UTC()
}

func NormalizeTaxID(taxID string) string {
	return strings.TrimSpace(taxID)
}

func ValidateNewParty(name, taxID string) error {
	if strings.TrimSpace(name) == "" {
		return ErrMissingPartyName
	}
	if NormalizeTaxID(taxID) == "" {
		return ErrMissingTaxID
	}
	return nil
}
