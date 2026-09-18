package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atta/internal/legalentities/application"
	"github.com/atta/internal/legalentities/application/commands"
	"github.com/atta/internal/legalentities/application/ports"
	"github.com/atta/internal/legalentities/application/queries"
	"github.com/atta/internal/legalentities/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	items []domain.LegalEntity
}

func (s *stubRepo) Count(ctx context.Context) (int, error) { return len(s.items), nil }
func (s *stubRepo) List(ctx context.Context) ([]domain.LegalEntity, error) {
	return s.items, nil
}
func (s *stubRepo) GetByID(ctx context.Context, id string) (*domain.LegalEntity, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			cp := s.items[i]
			return &cp, nil
		}
	}
	return nil, nil
}
func (s *stubRepo) Create(ctx context.Context, entity domain.LegalEntity) error {
	s.items = append(s.items, entity)
	return nil
}
func (s *stubRepo) Update(ctx context.Context, entity domain.LegalEntity) error {
	for i := range s.items {
		if s.items[i].ID == entity.ID {
			s.items[i] = entity
			return nil
		}
	}
	return nil
}

type stubPublisher struct{ calls int }

func (s *stubPublisher) PublishRegistered(ctx context.Context) error {
	s.calls++
	return nil
}

func testApp(repo ports.LegalEntityRepository, publisher ports.RegistrationPublisher) *application.Application {
	return &application.Application{
		Commands: application.Commands{
			CreateLegalEntity: commands.NewCreateLegalEntityCommand(repo, publisher),
			UpdateLegalEntity: commands.NewUpdateLegalEntityCommand(repo, publisher),
		},
		Queries: application.Queries{
			ListLegalEntities: queries.NewListLegalEntitiesQuery(repo),
		},
	}
}

func TestListLegalEntitiesEmpty(t *testing.T) {
	ctrl := NewController(testApp(&stubRepo{}, &stubPublisher{}))
	rr := httptest.NewRecorder()
	err := ctrl.ListLegalEntities(rr, httptest.NewRequest(http.MethodGet, "/api/v1/legal-entities", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)

	var body struct {
		Data []legalEntityResource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Empty(t, body.Data)
}

func TestCreateLegalEntity(t *testing.T) {
	repo := &stubRepo{}
	pub := &stubPublisher{}
	ctrl := NewController(testApp(repo, pub))
	payload := []byte(`{"data":{"attributes":{"tax_id":"900.123","scheme_id":"31","legal_name":"Acme SAS"}}}`)
	rr := httptest.NewRecorder()
	err := ctrl.CreateLegalEntity(rr, httptest.NewRequest(http.MethodPost, "/api/v1/legal-entities", bytes.NewReader(payload)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, 1, pub.calls)
	assert.Equal(t, "900123", repo.items[0].TaxID)
}

func TestCreateLegalEntityRejectsSecond(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubRepo{items: []domain.LegalEntity{{ID: "LE1", TaxID: "900", SchemeID: "31", LegalName: "A", CreatedAt: now, UpdatedAt: now}}}
	ctrl := NewController(testApp(repo, &stubPublisher{}))
	payload := []byte(`{"data":{"attributes":{"tax_id":"901","scheme_id":"31","legal_name":"B"}}}`)
	rr := httptest.NewRecorder()
	err := ctrl.CreateLegalEntity(rr, httptest.NewRequest(http.MethodPost, "/api/v1/legal-entities", bytes.NewReader(payload)))
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
}

func TestUpdateLegalEntity(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubRepo{items: []domain.LegalEntity{{ID: "LE1", TaxID: "9001", SchemeID: "31", LegalName: "Acme", CreatedAt: now, UpdatedAt: now}}}
	pub := &stubPublisher{}
	ctrl := NewController(testApp(repo, pub))
	payload := []byte(`{"data":{"attributes":{"legal_name":"Acme SAS"}}}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/legal-entities/LE1", bytes.NewReader(payload))
	req.SetPathValue("id", "LE1")
	rr := httptest.NewRecorder()
	err := ctrl.UpdateLegalEntity(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 0, pub.calls)
	assert.Equal(t, "Acme SAS", repo.items[0].LegalName)
}
