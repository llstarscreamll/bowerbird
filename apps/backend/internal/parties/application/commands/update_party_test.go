package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePartyCommand_RejectsEmptyRoles(t *testing.T) {
	repo := newMemoryPartyRepo()
	resolve := NewResolveOrCreateFromIssuerCommand(repo)
	resolve.newID = func() string { return "01PARTY000000000000000001" }
	party, err := resolve.Execute(context.Background(), "900", "Acme")
	require.NoError(t, err)

	update := NewUpdatePartyCommand(repo)
	empty := []string{}
	_, err = update.Execute(context.Background(), UpdatePartyInput{ID: party.ID, Roles: &empty})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeValidation, appErr.Code)
}

func TestUpdatePartyCommand_PreservesTaxID(t *testing.T) {
	repo := newMemoryPartyRepo()
	resolve := NewResolveOrCreateFromIssuerCommand(repo)
	resolve.newID = func() string { return "01PARTY000000000000000001" }
	party, err := resolve.Execute(context.Background(), "900", "Acme")
	require.NoError(t, err)

	update := NewUpdatePartyCommand(repo)
	name := "New Name"
	roles := []string{domain.RoleSupplier, domain.RoleCustomer}
	updated, err := update.Execute(context.Background(), UpdatePartyInput{ID: party.ID, Name: &name, Roles: &roles})
	require.NoError(t, err)
	assert.Equal(t, "900", updated.TaxID)
	assert.Equal(t, "New Name", updated.Name)
}
