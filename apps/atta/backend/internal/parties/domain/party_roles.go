package domain

import (
	"errors"
	"slices"
)

var (
	ErrMissingRoles = errors.New("party must have at least one role")
	ErrInvalidRole  = errors.New("invalid party role")
)

// PartyRoles is the validated role set for a party.
type PartyRoles struct {
	values []string
}

func ParsePartyRoles(raw []string) (PartyRoles, error) {
	if len(raw) == 0 {
		return PartyRoles{}, ErrMissingRoles
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, role := range raw {
		switch role {
		case RoleSupplier, RoleCustomer:
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		default:
			return PartyRoles{}, ErrInvalidRole
		}
	}
	if len(out) == 0 {
		return PartyRoles{}, ErrMissingRoles
	}
	slices.Sort(out)
	return PartyRoles{values: out}, nil
}

func (r PartyRoles) Strings() []string {
	if len(r.values) == 0 {
		return []string{}
	}
	out := make([]string, len(r.values))
	copy(out, r.values)
	return out
}

func (r PartyRoles) Equals(other PartyRoles) bool {
	if len(r.values) != len(other.values) {
		return false
	}
	for i := range r.values {
		if r.values[i] != other.values[i] {
			return false
		}
	}
	return true
}
