package domain

import (
	"strings"
	"time"
	"unicode"
)

const (
	SourceInvoice = "invoice"
	SourceManual  = "manual"

	EmailKindGeneral        = "general"
	EmailKindTaxMailbox     = "tax_mailbox"
	AddressKindPhysical     = "physical"
	AddressKindRegistration = "registration"
	AddressKindOther        = "other"

	TaxpayerKindLegal   = "1"
	TaxpayerKindNatural = "2"
)

type PartyEmail struct {
	ID        string
	Value     string
	Kind      string
	Source    string
	CreatedAt time.Time
}

type PartyPhone struct {
	ID         string
	Value      string
	Normalized string
	Source     string
	CreatedAt  time.Time
}

type PartyAddress struct {
	ID          string
	Line        string
	City        string
	Department  string
	PostalZone  string
	CountryCode string
	Kind        string
	Fingerprint string
	Source      string
	CreatedAt   time.Time
}

func NormalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func ClassifyEmailKind(value string) string {
	v := NormalizeEmail(value)
	at := strings.LastIndex(v, "@")
	local, host := v, ""
	if at >= 0 {
		local, host = v[:at], v[at+1:]
	}
	if strings.HasPrefix(local, "dte") || strings.Contains(host, "dte.") || strings.Contains(host, "facturaelectronica") {
		return EmailKindTaxMailbox
	}
	return EmailKindGeneral
}

func NormalizePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		return b.String()
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func ParseAddressKind(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	switch v {
	case "":
		return AddressKindOther, nil
	case AddressKindPhysical, AddressKindRegistration, AddressKindOther:
		return v, nil
	default:
		return "", ErrInvalidAddressKind
	}
}

func AddressFingerprint(line, city, department, country string) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(line)),
		strings.ToLower(strings.TrimSpace(city)),
		strings.ToLower(strings.TrimSpace(department)),
		strings.ToUpper(strings.TrimSpace(country)),
	}, "|")
}
