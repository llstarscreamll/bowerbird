package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssuerPartyLookup_NilSafe(t *testing.T) {
	lookup := NewIssuerPartyLookup(nil)
	id, err := lookup.ResolveIssuerPartyID(context.Background(), "900", "Acme")
	require.NoError(t, err)
	assert.Empty(t, id)
}

func TestNewIssuerPartyLookupFromApp_NilApp(t *testing.T) {
	lookup := NewIssuerPartyLookupFromApp(nil)
	id, err := lookup.ResolveIssuerPartyID(context.Background(), "900", "Acme")
	require.NoError(t, err)
	assert.Empty(t, id)
}
