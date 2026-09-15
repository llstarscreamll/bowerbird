package domain

import (
	"strings"
	"unicode"
)

// GTIN is a classified Global Trade Item Number (8/12/13/14 digits). No check digit.
type GTIN struct {
	value string
}

func ParseGTIN(raw string) (GTIN, error) {
	digits := gtinDigits(raw)
	switch len(digits) {
	case 8, 12, 13, 14:
		return GTIN{value: digits}, nil
	default:
		return GTIN{}, ErrInvalidGTIN
	}
}

func ClassifyGTIN(raw string) (GTIN, bool) {
	g, err := ParseGTIN(raw)
	if err != nil {
		return GTIN{}, false
	}
	return g, true
}

func gtinDigits(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == ' ' {
			continue
		}
		if !unicode.IsDigit(r) {
			return ""
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (g GTIN) String() string { return g.value }

func (g GTIN) Assigned() bool { return g.value != "" }
