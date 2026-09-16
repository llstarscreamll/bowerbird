package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingPartyName     = errors.New("missing party name")
	ErrMissingTaxID         = errors.New("missing party tax id")
	ErrPartyIDRequired      = errors.New("party id is required")
	ErrInvalidScheme        = errors.New("invalid identification scheme")
	ErrInvalidTaxpayerKind  = errors.New("invalid taxpayer kind")
	ErrInvalidEmail         = errors.New("invalid email")
	ErrMissingPhone         = errors.New("missing phone")
	ErrMissingAddress       = errors.New("missing address")
	ErrInvalidAddressKind   = errors.New("invalid address kind")
	ErrInvalidChannelSource = errors.New("invalid channel source")
	ErrDuplicateChannel     = errors.New("channel already exists")
	ErrChannelNotFound      = errors.New("channel not found")
)

const (
	RoleSupplier = "supplier"
	RoleCustomer = "customer"

	StatusProvisional = "provisional"
	StatusConfirmed   = "confirmed"

	CreationSourceManual  = "manual"
	CreationSourceInvoice = "invoice"
)

// Party is the trading-partner aggregate root (supplier/customer identity by tax id).
type Party struct {
	ID             string
	TaxID          string
	Name           string
	Roles          []string
	Status         string
	CreationSource string
	SchemeID       string
	TaxpayerKind   string
	TaxLevelCodes  []string
	Emails         []PartyEmail
	Phones         []PartyPhone
	Addresses      []PartyAddress
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
		ID:             id,
		TaxID:          taxID.String(),
		Name:           displayName,
		Roles:          []string{RoleSupplier},
		Status:         StatusProvisional,
		CreationSource: CreationSourceInvoice,
		TaxLevelCodes:  []string{},
		Emails:         []PartyEmail{},
		Phones:         []PartyPhone{},
		Addresses:      []PartyAddress{},
		CreatedAt:      now,
		UpdatedAt:      now,
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
		ID:             id,
		TaxID:          taxID.String(),
		Name:           displayName,
		Roles:          roles.Strings(),
		Status:         StatusConfirmed,
		CreationSource: CreationSourceManual,
		TaxLevelCodes:  []string{},
		Emails:         []PartyEmail{},
		Phones:         []PartyPhone{},
		Addresses:      []PartyAddress{},
		CreatedAt:      now,
		UpdatedAt:      now,
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

func (p *Party) FillScheme(raw string, now time.Time) (bool, error) {
	if strings.TrimSpace(p.SchemeID) != "" {
		return false, nil
	}
	scheme, err := ParseScheme(raw)
	if err != nil {
		return false, err
	}
	if scheme == "" {
		return false, nil
	}
	p.SchemeID = scheme
	p.UpdatedAt = now.UTC()
	return true, nil
}

func (p *Party) FillTaxpayerKind(raw string, now time.Time) (bool, error) {
	if strings.TrimSpace(p.TaxpayerKind) != "" {
		return false, nil
	}
	kind, err := ParseTaxpayerKind(raw)
	if err != nil {
		return false, err
	}
	if kind == "" {
		return false, nil
	}
	p.TaxpayerKind = kind
	p.UpdatedAt = now.UTC()
	return true, nil
}

func (p *Party) UnionTaxLevelCodes(codes []string, now time.Time) bool {
	if len(codes) == 0 {
		return false
	}
	seen := map[string]struct{}{}
	for _, c := range p.TaxLevelCodes {
		seen[c] = struct{}{}
	}
	changed := false
	for _, c := range codes {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		p.TaxLevelCodes = append(p.TaxLevelCodes, c)
		changed = true
	}
	if changed {
		p.UpdatedAt = now.UTC()
	}
	return changed
}

func parseChannelSource(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case SourceInvoice, SourceManual:
		return strings.TrimSpace(raw), nil
	default:
		return "", ErrInvalidChannelSource
	}
}

func (p *Party) AddEmail(id, raw, source string, now time.Time) (*PartyEmail, error) {
	value := NormalizeEmail(raw)
	if value == "" || !strings.Contains(value, "@") {
		return nil, ErrInvalidEmail
	}
	src, err := parseChannelSource(source)
	if err != nil {
		return nil, err
	}
	for _, existing := range p.Emails {
		if existing.Value == value {
			return nil, ErrDuplicateChannel
		}
	}
	kind := EmailKindGeneral
	if src == SourceInvoice {
		kind = ClassifyEmailKind(value)
	}
	now = now.UTC()
	email := PartyEmail{ID: id, Value: value, Kind: kind, Source: src, CreatedAt: now}
	p.Emails = append(p.Emails, email)
	p.UpdatedAt = now
	return &email, nil
}

func (p *Party) AddPhone(id, raw, source string, now time.Time) (*PartyPhone, error) {
	display := strings.TrimSpace(raw)
	normalized := NormalizePhone(display)
	if display == "" || normalized == "" {
		return nil, ErrMissingPhone
	}
	src, err := parseChannelSource(source)
	if err != nil {
		return nil, err
	}
	for _, existing := range p.Phones {
		if existing.Normalized == normalized {
			return nil, ErrDuplicateChannel
		}
	}
	now = now.UTC()
	phone := PartyPhone{ID: id, Value: display, Normalized: normalized, Source: src, CreatedAt: now}
	p.Phones = append(p.Phones, phone)
	p.UpdatedAt = now
	return &phone, nil
}

func (p *Party) AddAddress(id, line, city, department, postalZone, country, kind, source string, now time.Time) (*PartyAddress, error) {
	line, city, department = strings.TrimSpace(line), strings.TrimSpace(city), strings.TrimSpace(department)
	postalZone, country = strings.TrimSpace(postalZone), strings.TrimSpace(country)
	parsedKind, err := ParseAddressKind(kind)
	if err != nil {
		return nil, err
	}
	src, err := parseChannelSource(source)
	if err != nil {
		return nil, err
	}
	fp := AddressFingerprint(line, city, department, country)
	if fp == "|||" {
		return nil, ErrMissingAddress
	}
	for _, existing := range p.Addresses {
		if existing.Fingerprint == fp {
			return nil, ErrDuplicateChannel
		}
	}
	now = now.UTC()
	addr := PartyAddress{
		ID: id, Line: line, City: city, Department: department,
		PostalZone: postalZone, CountryCode: strings.ToUpper(country),
		Kind: parsedKind, Fingerprint: fp, Source: src, CreatedAt: now,
	}
	p.Addresses = append(p.Addresses, addr)
	p.UpdatedAt = now
	return &addr, nil
}

func (p *Party) RemoveEmail(id string, now time.Time) error {
	for i, e := range p.Emails {
		if e.ID == id {
			p.Emails = append(p.Emails[:i], p.Emails[i+1:]...)
			p.UpdatedAt = now.UTC()
			return nil
		}
	}
	return ErrChannelNotFound
}

func (p *Party) RemovePhone(id string, now time.Time) error {
	for i, e := range p.Phones {
		if e.ID == id {
			p.Phones = append(p.Phones[:i], p.Phones[i+1:]...)
			p.UpdatedAt = now.UTC()
			return nil
		}
	}
	return ErrChannelNotFound
}

func (p *Party) RemoveAddress(id string, now time.Time) error {
	for i, e := range p.Addresses {
		if e.ID == id {
			p.Addresses = append(p.Addresses[:i], p.Addresses[i+1:]...)
			p.UpdatedAt = now.UTC()
			return nil
		}
	}
	return ErrChannelNotFound
}

func NormalizeTaxID(taxID string) string {
	return strings.TrimSpace(taxID)
}
