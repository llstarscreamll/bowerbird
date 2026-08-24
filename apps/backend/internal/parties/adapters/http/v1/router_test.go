package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bowerbird/internal/parties/application"
	"github.com/bowerbird/internal/parties/application/commands"
	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/application/queries"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPartyRepo struct {
	parties map[string]domain.Party
}

func (s *stubPartyRepo) Create(ctx context.Context, party domain.Party) error {
	for _, existing := range s.parties {
		if existing.TaxID == party.TaxID && party.TaxID != "" {
			return appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
		}
	}
	s.parties[party.ID] = party
	return nil
}

func (s *stubPartyRepo) Update(ctx context.Context, party domain.Party) error {
	s.parties[party.ID] = party
	return nil
}

func (s *stubPartyRepo) GetByID(ctx context.Context, id string) (*domain.Party, error) {
	p, ok := s.parties[id]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (s *stubPartyRepo) GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error) {
	for _, p := range s.parties {
		if p.TaxID == taxID {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *stubPartyRepo) List(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	out := make([]domain.Party, 0)
	for _, p := range s.parties {
		if filter.Role != "" && !p.HasRole(filter.Role) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func testApp(repo ports.PartyRepository) *application.Application {
	return &application.Application{
		Commands: application.Commands{
			ResolveOrCreateFromIssuer: commands.NewResolveOrCreateFromIssuerCommand(repo),
			UpdateParty:               commands.NewUpdatePartyCommand(repo),
		},
		Queries: application.Queries{
			GetPartyByID: queries.NewGetPartyByIDQuery(repo),
			ListParties:  queries.NewListPartiesQuery(repo),
		},
	}
}

func TestListParties_FiltersByRole(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "Supplier", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
		"2": {ID: "2", TaxID: "901", Name: "Customer", Roles: []string{domain.RoleCustomer}, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	ctrl := NewController(testApp(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parties?role=supplier", nil)
	rr := httptest.NewRecorder()
	err := ctrl.ListParties(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)

	var body struct {
		Data []partyResource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "1", body.Data[0].ID)
}

func TestUpdateParty_DuplicateTaxIDConflictSurfaces(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "A", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
		"2": {ID: "2", TaxID: "901", Name: "B", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	err := repo.Create(context.Background(), domain.Party{ID: "3", TaxID: "900", Name: "Dup"})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
}
