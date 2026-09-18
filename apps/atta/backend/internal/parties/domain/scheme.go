package domain

import "strings"

const (
	SchemeNIT       = "31"
	SchemeCitizenID = "13"
)

func ParseScheme(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	switch v {
	case "":
		return "", nil
	case SchemeNIT, SchemeCitizenID:
		return v, nil
	default:
		return "", ErrInvalidScheme
	}
}

func ParseTaxpayerKind(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	switch v {
	case "":
		return "", nil
	case TaxpayerKindLegal, TaxpayerKindNatural:
		return v, nil
	default:
		return "", ErrInvalidTaxpayerKind
	}
}

func ParseTaxLevelCodes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		code := strings.ToUpper(strings.TrimSpace(p))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}
