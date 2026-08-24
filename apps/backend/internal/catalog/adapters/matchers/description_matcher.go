package matchers

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
)

// NormalizedDescriptionMatcher suggests items with an exact normalized description match.
// It never auto-links; callers treat results as suggestions only.
type NormalizedDescriptionMatcher struct {
	items ports.ItemRepository
}

func NewNormalizedDescriptionMatcher(items ports.ItemRepository) *NormalizedDescriptionMatcher {
	return &NormalizedDescriptionMatcher{items: items}
}

func (m *NormalizedDescriptionMatcher) Match(ctx context.Context, description string) ([]domain.Suggestion, error) {
	normalized := domain.NormalizeDescription(description)
	if normalized == "" || m.items == nil {
		return nil, nil
	}
	items, err := m.items.FindByNormalizedDescription(ctx, normalized)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Suggestion, 0, len(items))
	for _, item := range items {
		out = append(out, domain.Suggestion{
			ItemID: item.ID,
			Score:  1.0,
			Reason: "normalized_description_equality",
		})
	}
	return out, nil
}

var _ ports.SoftMatcher = (*NormalizedDescriptionMatcher)(nil)
