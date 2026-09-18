package application

import (
	"context"
	"testing"

	"github.com/atta/internal/legalentities/application/queries"
	"github.com/atta/internal/legalentities/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listStub struct {
	items []domain.LegalEntity
}

func (s listStub) Count(ctx context.Context) (int, error) { return len(s.items), nil }
func (s listStub) List(ctx context.Context) ([]domain.LegalEntity, error) {
	return s.items, nil
}
func (s listStub) GetByID(ctx context.Context, id string) (*domain.LegalEntity, error) {
	return nil, nil
}
func (s listStub) Create(ctx context.Context, entity domain.LegalEntity) error { return nil }
func (s listStub) Update(ctx context.Context, entity domain.LegalEntity) error { return nil }

func TestReceiverDirectoryMatchesNormalizedTaxID(t *testing.T) {
	dir := NewReceiverDirectory(&Application{
		Queries: Queries{
			ListLegalEntities: queries.NewListLegalEntitiesQuery(listStub{
				items: []domain.LegalEntity{{ID: "LE1", TaxID: "900123", SchemeID: "31", LegalName: "Acme"}},
			}),
		},
	})
	hasAny, err := dir.HasAny(context.Background())
	require.NoError(t, err)
	assert.True(t, hasAny)

	match, err := dir.ReceiverMatches(context.Background(), "900.123")
	require.NoError(t, err)
	assert.True(t, match)

	match, err = dir.ReceiverMatches(context.Background(), "900124")
	require.NoError(t, err)
	assert.False(t, match)
}
