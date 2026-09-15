package domain

import (
	"strings"
	"unicode"
)

// LineIdentifiers is UBL item identity on an invoice line (not catalog aliases).
type LineIdentifiers struct {
	BuyerCode string
	SellerSKU string
	GTIN      string
}

func ClassifyGTIN(raw string) (string, bool) {
	digits := gtinDigits(raw)
	switch len(digits) {
	case 8, 12, 13, 14:
		return digits, true
	default:
		return "", false
	}
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

// NewLineIdentifiers maps UBL buyer / seller / standard ids onto structured fields.
func NewLineIdentifiers(buyer, seller, standard string) LineIdentifiers {
	ids := LineIdentifiers{BuyerCode: strings.TrimSpace(buyer)}
	seller = strings.TrimSpace(seller)
	standard = strings.TrimSpace(standard)
	if gtin, ok := ClassifyGTIN(standard); ok {
		ids.GTIN = gtin
	}
	if seller != "" {
		ids.SellerSKU = seller
		return ids
	}
	if ids.GTIN == "" && standard != "" {
		ids.SellerSKU = standard
	}
	return ids
}

// FromCollapsedCode classifies a single legacy/LLM item_code (GTIN vs seller SKU).
func FromCollapsedCode(code string) LineIdentifiers {
	code = strings.TrimSpace(code)
	if gtin, ok := ClassifyGTIN(code); ok {
		return LineIdentifiers{GTIN: gtin}
	}
	return LineIdentifiers{SellerSKU: code}
}
