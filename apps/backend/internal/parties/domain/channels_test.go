package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyEmailKind(t *testing.T) {
	assert.Equal(t, EmailKindTaxMailbox, ClassifyEmailKind("dte_9002773704@dte.paperless.com.co"))
	assert.Equal(t, EmailKindGeneral, ClassifyEmailKind("infocolombia@ishopgroup.com"))
}

func TestAddChannelsUnionAndDedupe(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	taxID, err := ParseTaxID("900277370")
	require.NoError(t, err)
	party := NewProvisionalSupplier("P1", taxID, "I Shop", now)

	email, err := party.AddEmail("E1", "dte_900@dte.paperless.com.co", SourceInvoice, now)
	require.NoError(t, err)
	assert.Equal(t, EmailKindTaxMailbox, email.Kind)
	_, err = party.AddEmail("E2", "DTE_900@dte.paperless.com.co", SourceManual, now)
	assert.ErrorIs(t, err, ErrDuplicateChannel)

	phone, err := party.AddPhone("PH1", "(1) 3289133", SourceInvoice, now)
	require.NoError(t, err)
	assert.Equal(t, "13289133", phone.Normalized)
	_, err = party.AddPhone("PH2", "1 328 9133", SourceManual, now)
	assert.ErrorIs(t, err, ErrDuplicateChannel)

	addr, err := party.AddAddress("A1", "AV UNIVERSITARIA 50 21", "TUNJA", "BOYACÁ", "150001", "CO", AddressKindPhysical, SourceInvoice, now)
	require.NoError(t, err)
	assert.Equal(t, "av universitaria 50 21|tunja|boyacá|CO", addr.Fingerprint)
	_, err = party.AddAddress("A2", "AV UNIVERSITARIA 50 21", "Tunja", "BOYACÁ", "150001", "co", AddressKindRegistration, SourceInvoice, now)
	assert.ErrorIs(t, err, ErrDuplicateChannel)
}

func TestFillIdentityAndUnionTaxCodes(t *testing.T) {
	now := time.Now().UTC()
	taxID, err := ParseTaxID("900")
	require.NoError(t, err)
	party := NewProvisionalSupplier("P1", taxID, "Acme", now)

	changed, err := party.FillScheme("31", now)
	require.NoError(t, err)
	assert.True(t, changed)
	changed, err = party.FillScheme("13", now)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, SchemeNIT, party.SchemeID)

	changed, err = party.FillTaxpayerKind("1", now)
	require.NoError(t, err)
	assert.True(t, changed)
	changed, err = party.FillTaxpayerKind("2", now)
	require.NoError(t, err)
	assert.False(t, changed)

	assert.True(t, party.UnionTaxLevelCodes(ParseTaxLevelCodes("O-13;O-15"), now))
	assert.True(t, party.UnionTaxLevelCodes(ParseTaxLevelCodes("O-15;O-23"), now))
	assert.False(t, party.UnionTaxLevelCodes(ParseTaxLevelCodes("o-13;o-23"), now))
	assert.Equal(t, []string{"O-13", "O-15", "O-23"}, party.TaxLevelCodes)
}
