package domain

import "strings"

const (
	SchemeNIT       = "31"
	SchemeCitizenID = "13"
)

// IdentificationScheme is DIAN schemeName: 31 NIT or 13 cédula.
type IdentificationScheme struct {
	value string
}

func ParseScheme(raw string) (IdentificationScheme, error) {
	switch strings.TrimSpace(raw) {
	case SchemeNIT, SchemeCitizenID:
		return IdentificationScheme{value: strings.TrimSpace(raw)}, nil
	default:
		return IdentificationScheme{}, ErrInvalidScheme
	}
}

func (s IdentificationScheme) String() string { return s.value }

func (s IdentificationScheme) Equals(other IdentificationScheme) bool {
	return s.value == other.value
}
