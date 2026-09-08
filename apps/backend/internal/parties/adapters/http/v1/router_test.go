package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bowerbird/internal/parties/application"
	"github.com/bowerbird/internal/parties/application/commands"
	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/application/queries"
	"github.com/bowerbird/internal/parties/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPartyRepo struct {
	parties map[string]domain.Party
}

func (s *stubPartyRepo) Create(ctx context.Context, party domain.Party) error {
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
		if filter.CreationSource != "" && p.CreationSource != filter.CreationSource {
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
			CreateParty:               commands.NewCreatePartyCommand(repo),
			UpdateParty:               commands.NewUpdatePartyCommand(repo),
		},
		Queries: application.Queries{
			GetPartyByID: queries.NewGetPartyByIDQuery(repo),
			ListParties:  queries.NewListPartiesQuery(repo),
		},
	}
}

func TestListParties_FiltersByCreationSource(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "From Invoice", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
		"2": {ID: "2", TaxID: "901", Name: "Manual", Roles: []string{domain.RoleCustomer}, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, CreatedAt: now, UpdatedAt: now},
	}}
	ctrl := NewController(testApp(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parties?creation_source=invoice", nil)
	rr := httptest.NewRecorder()
	err := ctrl.ListParties(rr, req)
	require.NoError(t, err)

	var body struct {
		Data []partyResource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "1", body.Data[0].ID)
	assert.Equal(t, domain.CreationSourceInvoice, body.Data[0].Attributes.CreationSource)
}
