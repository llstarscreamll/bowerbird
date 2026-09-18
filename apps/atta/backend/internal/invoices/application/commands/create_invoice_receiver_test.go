package commands

import (
	"context"
	"testing"

	"github.com/atta/internal/invoices/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInvoice_SkipsWhenNoLegalEntity(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &lineResolverStub{}, receiverDirectoryStub{})
	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-SKIP-1",
			InvoiceID: "FV-SKIP-1",
			Issuer:    domain.Party{Name: "Proveedor", TaxID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", TaxID: "901456"},
			Lines:     []domain.InvoiceLine{{LineID: "1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, result)
	assert.False(t, repo.persisted)
}

func TestCreateInvoice_SkipsWhenReceiverDoesNotMatch(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &lineResolverStub{}, receiverDirectoryStub{hasAny: true, match: false})
	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-SKIP-2",
			InvoiceID: "FV-SKIP-2",
			Issuer:    domain.Party{Name: "Proveedor", TaxID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", TaxID: "901456"},
			Lines:     []domain.InvoiceLine{{LineID: "1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, result)
	assert.False(t, repo.persisted)
}

func TestCreateInvoice_PersistsWhenReceiverMatches(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &lineResolverStub{}, matchingReceivers())
	cmd.newID = func() string { return "ID-1" }
	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-MATCH-1",
			InvoiceID: "FV-MATCH-1",
			Issuer:    domain.Party{Name: "Proveedor", TaxID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", TaxID: "901456"},
			Lines:     []domain.InvoiceLine{{LineID: "1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		},
		SourceName:       "test",
		SourceID:         "src-match-1",
		ExtractionSource: "xml",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, repo.persisted)
}
