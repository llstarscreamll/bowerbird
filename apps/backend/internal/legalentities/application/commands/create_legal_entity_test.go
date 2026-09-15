package commands

import (
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/legalentities/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	items     []domain.LegalEntity
	createErr error
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
	if s.createErr != nil {
		return s.createErr
	}
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

type stubPublisher struct {
	calls int
}

func (s *stubPublisher) PublishRegistered(ctx context.Context) error {
	s.calls++
	return nil
}

func TestCreateLegalEntityRejectsSecond(t *testing.T) {
	repo := &stubRepo{items: []domain.LegalEntity{{ID: "LE1", TaxID: "900", SchemeID: "31", LegalName: "A"}}}
	cmd := NewCreateLegalEntityCommand(repo, &stubPublisher{})
	_, err := cmd.Execute(context.Background(), CreateLegalEntityInput{TaxID: "901", SchemeID: "31", LegalName: "B"})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
}

func TestCreateLegalEntityPublishes(t *testing.T) {
	repo := &stubRepo{}
	pub := &stubPublisher{}
	cmd := NewCreateLegalEntityCommand(repo, pub)
	cmd.now = func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }
	cmd.newID = func() string { return "LE-NEW" }

	entity, err := cmd.Execute(context.Background(), CreateLegalEntityInput{TaxID: "900.1", SchemeID: "31", LegalName: "Acme"})
	require.NoError(t, err)
	assert.Equal(t, "9001", entity.TaxID)
	assert.Equal(t, 1, pub.calls)
}

func TestUpdateLegalEntityPublishesOnTaxIDChange(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	repo := &stubRepo{items: []domain.LegalEntity{{ID: "LE1", TaxID: "9001", SchemeID: "31", LegalName: "Acme", CreatedAt: now, UpdatedAt: now}}}
	pub := &stubPublisher{}
	cmd := NewUpdateLegalEntityCommand(repo, pub)
	tax := "9002"
	entity, err := cmd.Execute(context.Background(), UpdateLegalEntityInput{ID: "LE1", TaxID: &tax})
	require.NoError(t, err)
	assert.Equal(t, "9002", entity.TaxID)
	assert.Equal(t, 1, pub.calls)
}
