package v1

import (
	"bytes"
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

func TestListParties_FiltersByRole(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "Supplier", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
		"2": {ID: "2", TaxID: "901", Name: "Customer", Roles: []string{domain.RoleCustomer}, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, CreatedAt: now, UpdatedAt: now},
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

func TestCreateParty_Returns201(t *testing.T) {
	repo := &stubPartyRepo{parties: map[string]domain.Party{}}
	ctrl := NewController(testApp(repo))
	body := `{"data":{"attributes":{"name":"Acme","tax_id":"900123","roles":["customer"]}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parties", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	err := ctrl.CreateParty(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp struct {
		Data partyResource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "parties", resp.Data.Type)
	assert.Equal(t, "900123", resp.Data.Attributes.TaxID)
	assert.Equal(t, domain.StatusConfirmed, resp.Data.Attributes.Status)
	assert.Equal(t, domain.CreationSourceManual, resp.Data.Attributes.CreationSource)
}

func TestCreateParty_DuplicateTaxID409(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "Existing", Roles: []string{domain.RoleSupplier}, Status: domain.StatusConfirmed, CreatedAt: now, UpdatedAt: now},
	}}
	ctrl := NewController(testApp(repo))
	body := `{"data":{"attributes":{"name":"Other","tax_id":"900","roles":["customer"]}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parties", bytes.NewBufferString(body))
	err := ctrl.CreateParty(httptest.NewRecorder(), req)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
}

func TestCreateParty_EmptyRoles422(t *testing.T) {
	repo := &stubPartyRepo{parties: map[string]domain.Party{}}
	ctrl := NewController(testApp(repo))
	body := `{"data":{"attributes":{"name":"Acme","tax_id":"900","roles":[]}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parties", bytes.NewBufferString(body))
	err := ctrl.CreateParty(httptest.NewRecorder(), req)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeValidation, appErr.Code)
}

func TestUpdateParty_PreservesTaxID(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubPartyRepo{parties: map[string]domain.Party{
		"1": {ID: "1", TaxID: "900", Name: "A", Roles: []string{domain.RoleSupplier}, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	ctrl := NewController(testApp(repo))
	body := `{"data":{"attributes":{"name":"Updated","tax_id":"999","roles":["supplier","customer"]}}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/parties/1", bytes.NewBufferString(body))
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	err := ctrl.UpdateParty(rr, req)
	require.NoError(t, err)

	var resp struct {
		Data partyResource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "900", resp.Data.Attributes.TaxID)
	assert.Equal(t, "Updated", resp.Data.Attributes.Name)
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
