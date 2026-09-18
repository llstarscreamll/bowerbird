package domain

import "strings"

// HardHits collects independent hard-identity matches for one invoice line.
type HardHits struct {
	BuyerItemID  string
	GTINItemID   string
	SellerItemID string
}

// Agree returns the shared item id, or conflict with the distinct ids (stable order: buyer, gtin, seller).
func (h HardHits) Agree() (itemID string, conflict bool, ids []string) {
	seen := make([]string, 0, 3)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		for _, existing := range seen {
			if existing == id {
				return
			}
		}
		seen = append(seen, id)
	}
	add(h.BuyerItemID)
	add(h.GTINItemID)
	add(h.SellerItemID)
	switch len(seen) {
	case 0:
		return "", false, nil
	case 1:
		return seen[0], false, seen
	default:
		return "", true, seen
	}
}
