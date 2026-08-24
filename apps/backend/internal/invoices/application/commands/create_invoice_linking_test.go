package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type linkingRepoStub struct {
	persisted bool
	applied   bool
	status    string
	partyID   *string
	lines     []ports.LineLinkUpdate
}

func (r *linkingRepoStub) PersistInvoiceAtomic(ctx context.Context, header domain.InvoiceHeaderRecord, lines []domain.InvoiceLineRecord) error {
	r.persisted = true
	return nil
}

func (r *linkingRepoStub) ApplyCatalogLinking(ctx context.Context, headerID string, issuerPartyID *string, linkingStatus string, lines []ports.LineLinkUpdate) error {
	r.applied = true
	r.status = linkingStatus
	r.partyID = issuerPartyID
	r.lines = lines
	return nil
}

type partyResolverStub struct {
	id string
}

func (p *partyResolverStub) ResolveIssuerPartyID(ctx context.Context, taxID, name string) (string, error) {
	return p.id, nil
}

type lineResolverStub struct{}

func (l *lineResolverStub) ResolveLine(ctx context.Context, input ports.CatalogLineResolveInput) (*ports.CatalogLineResolveResult, error) {
	if input.ItemCode == "" {
		return &ports.CatalogLineResolveResult{Status: "unmatched"}, nil
	}
	suggestions, _ := json.Marshal([]any{})
	return &ports.CatalogLineResolveResult{
		ItemID:      "ITEM-PROV",
		Status:      "linked",
		Method:      "hard",
		Suggestions: suggestions,
	}, nil
}

type failingLineResolver struct{}

func (l *failingLineResolver) ResolveLine(ctx context.Context, input ports.CatalogLineResolveInput) (*ports.CatalogLineResolveResult, error) {
	return nil, assert.AnError
}

func TestCreateInvoice_LinksPartyAndProvisionalItem(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &lineResolverStub{})
	n := 0
	cmd.newID = func() string {
		n++
		return "ID-" + string(rune('0'+n))
	}

	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-LINK-1",
			InvoiceID: "FV-1",
			Issuer:    domain.Party{Name: "Proveedor", CompanyID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", CompanyID: "901456"},
			Lines: []domain.InvoiceLine{
				{LineID: "1", ItemCode: "SKU-1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10},
				{LineID: "2", ItemCode: "", ItemDescription: "Service", Quantity: 1, UnitPrice: 5, LineExtension: 5},
			},
		},
		SourceName:       "test",
		SourceID:         "src-1",
		ExtractionSource: "xml",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, repo.persisted)
	assert.True(t, repo.applied)
	assert.Equal(t, "linked", repo.status)
	require.NotNil(t, repo.partyID)
	assert.Equal(t, "PARTY-1", *repo.partyID)
	require.Len(t, repo.lines, 2)
	assert.Equal(t, "linked", repo.lines[0].LinkStatus)
	assert.Equal(t, "unmatched", repo.lines[1].LinkStatus)
}

func TestCreateInvoice_LinkFailureKeepsPersistedInvoice(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &failingLineResolver{})
	n := 0
	cmd.newID = func() string {
		n++
		return "ID-" + string(rune('0'+n))
	}

	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-LINK-2",
			InvoiceID: "FV-2",
			Issuer:    domain.Party{Name: "Proveedor", CompanyID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", CompanyID: "901456"},
			Lines:     []domain.InvoiceLine{{LineID: "1", ItemCode: "SKU-1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10}},
		},
		SourceName:       "test",
		SourceID:         "src-2",
		ExtractionSource: "xml",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, repo.persisted)
	assert.True(t, repo.applied)
	assert.Equal(t, "failed", repo.status)
	require.NotNil(t, repo.partyID)
	assert.Equal(t, "PARTY-1", *repo.partyID)
	assert.Empty(t, repo.lines)
}

type partialFailLineResolver struct {
	calls int
}

func (l *partialFailLineResolver) ResolveLine(ctx context.Context, input ports.CatalogLineResolveInput) (*ports.CatalogLineResolveResult, error) {
	l.calls++
	if l.calls == 1 {
		return &ports.CatalogLineResolveResult{
			ItemID:      "ITEM-1",
			Status:      "linked",
			Method:      "hard",
			Suggestions: []byte("[]"),
		}, nil
	}
	return nil, assert.AnError
}

func TestCreateInvoice_PartialLineLinkingPersistsSuccessfulLines(t *testing.T) {
	repo := &linkingRepoStub{}
	cmd := NewCreateInvoiceCommand(repo, &partyResolverStub{id: "PARTY-1"}, &partialFailLineResolver{})
	n := 0
	cmd.newID = func() string {
		n++
		return "ID-" + string(rune('0'+n))
	}

	result, err := cmd.Execute(context.Background(), CreateInvoiceInput{
		Invoice: &domain.InvoiceDocument{
			CUFE:      "CUFE-LINK-3",
			InvoiceID: "FV-3",
			Issuer:    domain.Party{Name: "Proveedor", CompanyID: "900123"},
			Receiver:  domain.Party{Name: "Cliente", CompanyID: "901456"},
			Lines: []domain.InvoiceLine{
				{LineID: "1", ItemCode: "SKU-1", ItemDescription: "Widget", Quantity: 1, UnitPrice: 10, LineExtension: 10},
				{LineID: "2", ItemCode: "SKU-2", ItemDescription: "Gadget", Quantity: 1, UnitPrice: 5, LineExtension: 5},
			},
		},
		SourceName:       "test",
		SourceID:         "src-3",
		ExtractionSource: "xml",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "failed", repo.status)
	require.NotNil(t, repo.partyID)
	assert.Equal(t, "PARTY-1", *repo.partyID)
	require.Len(t, repo.lines, 1)
	assert.Equal(t, "linked", repo.lines[0].LinkStatus)
	assert.Equal(t, "ITEM-1", *repo.lines[0].ItemID)
}
