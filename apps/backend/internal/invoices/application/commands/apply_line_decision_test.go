package commands

import (
	"context"
	"testing"

	"github.com/bowerbird/internal/invoices/application/ports"
	invoicesDomain "github.com/bowerbird/internal/invoices/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLineLinks struct {
	lines map[string]*invoicesDomain.LineForDecision
}

func (s *stubLineLinks) SaveLineLink(ctx context.Context, lineID string, link invoicesDomain.LineLink) error {
	line := s.lines[lineID]
	if line == nil {
		return nil
	}
	line.Link = link
	return nil
}

func (s *stubLineLinks) ListReviewLines(ctx context.Context, statuses []string) ([]ports.ReviewLine, error) {
	return nil, nil
}

func (s *stubLineLinks) GetLineForDecision(ctx context.Context, lineID string) (*invoicesDomain.LineForDecision, error) {
	line, ok := s.lines[lineID]
	if !ok {
		return nil, nil
	}
	cp := *line
	return &cp, nil
}

func (s *stubLineLinks) SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error {
	return nil
}

type stubCatalogMatching struct {
	items       map[string]bool
	mint        string
	memoryErr   error
	memoryCalls int
}

func (s *stubCatalogMatching) ValidateItemExists(ctx context.Context, itemID string) error {
	if !s.items[itemID] {
		return appErrors.New(appErrors.CodeNotFound, "catalog item not found")
	}
	return nil
}

func (s *stubCatalogMatching) MintProvisionalFromEvidence(ctx context.Context, input ports.MintProvisionalInput) (string, error) {
	if s.mint != "" {
		return s.mint, nil
	}
	return "ITEM-NEW", nil
}

func (s *stubCatalogMatching) EnsureSupplierAlias(ctx context.Context, partyID, itemCode, itemID string) error {
	return nil
}

func (s *stubCatalogMatching) RecordMatchMemory(ctx context.Context, input ports.MatchMemoryInput) error {
	s.memoryCalls++
	return s.memoryErr
}

func TestApplyLineDecision_LinkRemember(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				PartyID:         "P1",
				ItemCode:        "SKU-1",
				Description:     "Widget",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	catalog := &stubCatalogMatching{items: map[string]bool{"ITEM-1": true}}
	cmd := NewApplyLineDecisionCommand(links, catalog)

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "INV-1",
		LineID:    "LINE-1",
		ItemID:    "ITEM-1",
		Action:    invoicesDomain.MemoryActionLink,
		Remember:  true,
		Lock:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, invoicesDomain.LinkStatusLinked, links.lines["LINE-1"].Link.Status)
	assert.True(t, links.lines["LINE-1"].Link.Locked)
}

func TestApplyLineDecision_Reject(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	cmd := NewApplyLineDecisionCommand(links, &stubCatalogMatching{})

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "INV-1",
		LineID:    "LINE-1",
		Action:    invoicesDomain.MemoryActionNeverMatch,
		Remember:  true,
		Lock:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, invoicesDomain.LinkStatusRejected, links.lines["LINE-1"].Link.Status)
}

func TestApplyLineDecision_CreateProvisional(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				PartyID:         "P1",
				ItemCode:        "SKU-1",
				Description:     "Widget",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	catalog := &stubCatalogMatching{items: map[string]bool{"ITEM-NEW": true}, mint: "ITEM-NEW"}
	cmd := NewApplyLineDecisionCommand(links, catalog)

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "INV-1",
		LineID:    "LINE-1",
		Action:    invoicesDomain.ActionCreateProvisional,
	})
	require.NoError(t, err)
	require.NotNil(t, links.lines["LINE-1"].Link.ItemID)
	assert.Equal(t, "ITEM-NEW", *links.lines["LINE-1"].Link.ItemID)
	assert.True(t, links.lines["LINE-1"].Link.Locked)
}

func TestApplyLineDecision_ValidateFailsWithoutMutatingLine(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	cmd := NewApplyLineDecisionCommand(links, &stubCatalogMatching{items: map[string]bool{}})

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "INV-1",
		LineID:    "LINE-1",
		ItemID:    "MISSING",
		Action:    invoicesDomain.MemoryActionLink,
	})
	require.Error(t, err)
	assert.Equal(t, invoicesDomain.LinkStatusUnmatched, links.lines["LINE-1"].Link.Status)
}

func TestApplyLineDecision_MemoryFailsWithoutMutatingLine(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				PartyID:         "P1",
				ItemCode:        "SKU-1",
				Description:     "Widget",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	catalog := &stubCatalogMatching{
		items:     map[string]bool{"ITEM-1": true},
		memoryErr: appErrors.New(appErrors.CodeInternal, "memory write failed"),
	}
	cmd := NewApplyLineDecisionCommand(links, catalog)

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "INV-1",
		LineID:    "LINE-1",
		ItemID:    "ITEM-1",
		Action:    invoicesDomain.MemoryActionLink,
		Remember:  true,
	})
	require.Error(t, err)
	assert.Equal(t, 1, catalog.memoryCalls)
	assert.Equal(t, invoicesDomain.LinkStatusUnmatched, links.lines["LINE-1"].Link.Status)
}

func TestApplyLineDecision_InvoiceMismatch(t *testing.T) {
	links := &stubLineLinks{
		lines: map[string]*invoicesDomain.LineForDecision{
			"LINE-1": {
				LineID:          "LINE-1",
				InvoiceHeaderID: "INV-1",
				Link:            invoicesDomain.LineLink{Status: invoicesDomain.LinkStatusUnmatched},
			},
		},
	}
	cmd := NewApplyLineDecisionCommand(links, &stubCatalogMatching{items: map[string]bool{"ITEM-1": true}})

	err := cmd.Execute(context.Background(), ApplyLineDecisionInput{
		InvoiceID: "OTHER",
		LineID:    "LINE-1",
		ItemID:    "ITEM-1",
		Action:    invoicesDomain.MemoryActionLink,
	})
	require.Error(t, err)
	assert.Equal(t, invoicesDomain.LinkStatusUnmatched, links.lines["LINE-1"].Link.Status)
}
