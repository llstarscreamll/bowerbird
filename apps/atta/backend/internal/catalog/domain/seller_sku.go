package domain

import "strings"

var unusableSellerTokens = map[string]struct{}{
	"1": {}, "01": {}, "001": {},
	"n/a": {}, "na": {},
	"serv": {}, "servicio": {}, "item": {},
}

// SellerSKU is a supplier-issued product code on an invoice line.
type SellerSKU struct {
	value string
}

func ParseSellerSKU(raw string) SellerSKU {
	return SellerSKU{value: strings.TrimSpace(raw)}
}

func (s SellerSKU) String() string { return s.value }

func (s SellerSKU) Assigned() bool { return s.value != "" }

func (s SellerSKU) Usable() bool { return SellerSKUUsable(s.value) }

// SellerSKUUsable reports whether a seller SKU may be auto-learned or used to mint.
func SellerSKUUsable(code string) bool {
	return ParseSellerSKU(code).usable()
}

func (s SellerSKU) usable() bool {
	if len(s.value) < 3 {
		return false
	}
	_, blocked := unusableSellerTokens[strings.ToLower(s.value)]
	return !blocked
}

func UsableSellerSKU(code string) string {
	s := ParseSellerSKU(code)
	if !s.Usable() {
		return ""
	}
	return s.String()
}
