package domain

import (
	"strings"
	"unicode"
)

// TaxID is the tenant's own fiscal identifier (digits only).
type TaxID struct {
	value string
}

func NormalizeTaxID(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ParseTaxID(raw string) (TaxID, error) {
	v := NormalizeTaxID(raw)
	if v == "" {
		return TaxID{}, ErrMissingTaxID
	}
	return TaxID{value: v}, nil
}

func (t TaxID) String() string { return t.value }

func (t TaxID) Equals(other TaxID) bool { return t.value == other.value }

func (t TaxID) Matches(raw string) bool {
	other, err := ParseTaxID(raw)
	if err != nil {
		return false
	}
	return t.Equals(other)
}
