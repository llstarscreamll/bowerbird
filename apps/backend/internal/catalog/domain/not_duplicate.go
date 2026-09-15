package domain

import "strings"

// NotDuplicatePair is a steward decision that two items are not the same product.
type NotDuplicatePair struct {
	Left  string
	Right string
}

func NewNotDuplicatePair(a, b string) (NotDuplicatePair, error) {
	left := strings.TrimSpace(a)
	right := strings.TrimSpace(b)
	if left == "" || right == "" {
		return NotDuplicatePair{}, ErrItemIDRequired
	}
	if left == right {
		return NotDuplicatePair{}, ErrCannotMergeIntoSelf
	}
	if left > right {
		left, right = right, left
	}
	return NotDuplicatePair{Left: left, Right: right}, nil
}

func NotDuplicatePairsFromIDs(ids []string) ([]NotDuplicatePair, error) {
	unique := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) < 2 {
		return nil, ErrItemIDRequired
	}
	out := make([]NotDuplicatePair, 0, len(unique)*(len(unique)-1)/2)
	for i := 0; i < len(unique); i++ {
		for j := i + 1; j < len(unique); j++ {
			pair, err := NewNotDuplicatePair(unique[i], unique[j])
			if err != nil {
				return nil, err
			}
			out = append(out, pair)
		}
	}
	return out, nil
}
