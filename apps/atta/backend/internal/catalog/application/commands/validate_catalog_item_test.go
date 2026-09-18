package commands

import (
	"context"
	"testing"
	"time"

	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCatalogItemRejectsMerged(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	cmd := NewValidateCatalogItemCommand(&memItems{items: map[string]domain.Item{
		"B": {
			ID:           "B",
			Name:         "Merged",
			Kind:         domain.KindGoods,
			Status:       domain.StatusMerged,
			MergedIntoID: "A",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}})
	err := cmd.Execute(context.Background(), "B")
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeGone, appErr.Code)
	assert.Equal(t, "A", appErr.Meta["merged_into_id"])
}

func TestValidateCatalogItemAcceptsActive(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	cmd := NewValidateCatalogItemCommand(&memItems{items: map[string]domain.Item{
		"A": testItem("A", "Widget", "INT-A", now),
	}})
	require.NoError(t, cmd.Execute(context.Background(), "A"))
}
