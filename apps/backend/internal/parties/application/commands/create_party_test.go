package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePartyCommand_CreatesConfirmedParty(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewCreatePartyCommand(repo)
	cmd.now = func() time.Time { return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC) }
	cmd.newID = func() string { return "01PARTY000000000000000010" }

	party, err := cmd.Execute(context.Background(), CreatePartyInput{
		Name:  "Acme",
		TaxID: "900123",
		Roles: []string{domain.RoleCustomer},
	})
	require.NoError(t, err)
	require.NotNil(t, party)
	assert.Equal(t, domain.StatusConfirmed, party.Status)
	assert.Equal(t, "900123", party.TaxID)
}

func TestCreatePartyCommand_DuplicateTaxID(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewCreatePartyCommand(repo)
	cmd.newID = func() string { return "01PARTY000000000000000010" }

	_, err := cmd.Execute(context.Background(), CreatePartyInput{
		Name: "First", TaxID: "900", Roles: []string{domain.RoleSupplier},
	})
	require.NoError(t, err)

	_, err = cmd.Execute(context.Background(), CreatePartyInput{
		Name: "Second", TaxID: "900", Roles: []string{domain.RoleCustomer},
	})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
}
