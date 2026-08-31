package domain

// TaxID is the validated fiscal identifier (NIT) for a party.
type TaxID struct {
	value string
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
