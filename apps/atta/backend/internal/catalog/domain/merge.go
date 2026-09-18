package domain

import "strings"

// UnionAliases plans the alias rewrite for a merge: colliding tuples on the
// surviving item are dropped; others are reassigned. Source and party stay.
func UnionAliases(survivorID string, held, incoming []Alias) (reassign []Alias, deleteIDs []string, err error) {
	survivorID = strings.TrimSpace(survivorID)
	if survivorID == "" {
		return nil, nil, ErrItemIDRequired
	}
	have := map[string]struct{}{}
	for _, alias := range held {
		have[aliasTupleKey(alias)] = struct{}{}
	}
	for _, alias := range incoming {
		key := aliasTupleKey(alias)
		if _, ok := have[key]; ok {
			deleteIDs = append(deleteIDs, alias.ID)
			continue
		}
		if err := alias.ReassignTo(survivorID); err != nil {
			return nil, nil, err
		}
		reassign = append(reassign, alias)
		have[key] = struct{}{}
	}
	return reassign, deleteIDs, nil
}

func aliasTupleKey(alias Alias) string {
	party := ""
	if alias.PartyID != nil {
		party = *alias.PartyID
	}
	return alias.Scheme + "|" + party + "|" + alias.Value
}

// PickMergeName keeps the surviving name unless it is an invoice mint with a poorer label.
func PickMergeName(survivor Item, sources []Item) string {
	if survivor.CreationSource != CreationSourceInvoice {
		return survivor.Name
	}
	best := survivor.Name
	for _, source := range sources {
		if len(source.Name) > len(best) {
			best = source.Name
		}
	}
	return best
}

// PickMergeKind keeps a classified kind; otherwise the first non-unknown source.
func PickMergeKind(survivor Item, sources []Item) string {
	if survivor.Kind != KindUnknown && survivor.Kind != "" {
		return survivor.Kind
	}
	for _, source := range sources {
		if source.Kind != KindUnknown && source.Kind != "" {
			return source.Kind
		}
	}
	return survivor.Kind
}
