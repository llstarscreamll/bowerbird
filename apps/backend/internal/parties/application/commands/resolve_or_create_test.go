package commands

import (
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryPartyRepo struct {
	byID    map[string]domain.Party
	byTaxID map[string]string
}

func newMemoryPartyRepo() *memoryPartyRepo {
	return &memoryPartyRepo{byID: map[string]domain.Party{}, byTaxID: map[string]string{}}
}

func (r *memoryPartyRepo) Create(ctx context.Context, party domain.Party) error {
	r.byID[party.ID] = party
	r.byTaxID[party.TaxID] = party.ID
	return nil
}

func (r *memoryPartyRepo) Update(ctx context.Context, party domain.Party) error {
	r.byID[party.ID] = party
	r.byTaxID[party.TaxID] = party.ID
	return nil
}

func (r *memoryPartyRepo) GetByID(ctx context.Context, id string) (*domain.Party, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (r *memoryPartyRepo) GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error) {
	id, ok := r.byTaxID[taxID]
	if !ok {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *memoryPartyRepo) List(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	out := make([]domain.Party, 0, len(r.byID))
	for _, p := range r.byID {
		out = append(out, p)
	}
	return out, nil
}

func TestResolveOrCreateFromIssuer_CreatesProvisionalSupplier(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewResolveOrCreateFromIssuerCommand(repo)
	cmd.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC) }
	cmd.newID = func() string { return "01PARTY000000000000000001" }

	party, err := cmd.Execute(context.Background(), "900123", "Proveedor SA")
	require.NoError(t, err)
	require.NotNil(t, party)
	assert.Equal(t, "900123", party.TaxID)
	assert.Equal(t, "Proveedor SA", party.Name)
	assert.Equal(t, domain.StatusProvisional, party.Status)
	assert.True(t, party.HasRole(domain.RoleSupplier))
}

func TestResolveOrCreateFromIssuer_ReusesExisting(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewResolveOrCreateFromIssuerCommand(repo)
	cmd.newID = func() string { return "01PARTY000000000000000001" }

	first, err := cmd.Execute(context.Background(), "900123", "Proveedor")
	require.NoError(t, err)
	cmd.newID = func() string { return "01PARTY000000000000000002" }
	second, err := cmd.Execute(context.Background(), "900123", "Other Name")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, repo.byID, 1)
}

func TestResolveOrCreateFromIssuer_EmptyTaxID(t *testing.T) {
	cmd := NewResolveOrCreateFromIssuerCommand(newMemoryPartyRepo())
	party, err := cmd.Execute(context.Background(), "  ", "Name")
	require.NoError(t, err)
	assert.Nil(t, party)
}
