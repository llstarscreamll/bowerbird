package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIssuerPartyLookupPanicsWhenCommandMissing(t *testing.T) {
	assert.Panics(t, func() { NewIssuerPartyLookup(nil) })
}

func TestNewIssuerPartyLookupFromAppPanicsWhenAppMissing(t *testing.T) {
	assert.Panics(t, func() { NewIssuerPartyLookupFromApp(nil) })
}
