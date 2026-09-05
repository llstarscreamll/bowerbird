package domain

import (
	"errors"
	"strings"
)

var (
	ErrItemIDRequired = errors.New("item id is required for link")
	ErrLineLinkLocked = errors.New("line link is locked")
	ErrInvalidAction  = errors.New("invalid link decision action")
)

const (
	LinkStatusUnmatched = "unmatched"
	LinkStatusSuggested = "suggested"
	LinkStatusLinked    = "linked"
	LinkStatusRejected  = "rejected"

	LinkMethodMemory = "memory"
	LinkMethodHard   = "hard"
	LinkMethodSoft   = "soft"
	LinkMethodManual = "manual"

	MemoryActionLink        = "link"
	MemoryActionNeverMatch  = "never_match"
	ActionCreateProvisional = "create_provisional"

	LinkingStatusPending = "pending"
	LinkingStatusLinked  = "linked"
	LinkingStatusFailed  = "failed"
)

var emptySuggestions = []byte("[]")

// LineLink is the catalog link state for an invoice line (value object).
type LineLink struct {
	ItemID      *string
	Status      string
	Method      string
	Locked      bool
	Suggestions []byte
}

func (l LineLink) IsLinked() bool {
	return l.Status == LinkStatusLinked
}

// LineForDecision is the context needed to apply a manual catalog link decision.
type LineForDecision struct {
	LineID          string
	InvoiceHeaderID string
	ItemCode        string
	Description     string
	PartyID         string
	Link            LineLink
}

func (l LineForDecision) BelongsToInvoice(invoiceID string) bool {
	id := strings.TrimSpace(invoiceID)
	return id != "" && l.InvoiceHeaderID == id
}

// ApplyManualDecision applies link | never_match to this LineLink.
func (l LineLink) ApplyManualDecision(action, itemID string, lock bool) (LineLink, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		action = MemoryActionLink
	}
	switch action {
	case MemoryActionLink:
		return l.ApplyManualLink(itemID, lock)
	case MemoryActionNeverMatch:
		return l.Reject(lock)
	default:
		return LineLink{}, ErrInvalidAction
	}
}

func (l LineLink) ApplyManualLink(itemID string, lock bool) (LineLink, error) {
	if l.Locked {
		return LineLink{}, ErrLineLinkLocked
	}
	idCopy := strings.TrimSpace(itemID)
	if idCopy == "" {
		return LineLink{}, ErrItemIDRequired
	}
	return LineLink{
		ItemID:      &idCopy,
		Status:      LinkStatusLinked,
		Method:      LinkMethodManual,
		Locked:      lock,
		Suggestions: emptySuggestions,
	}, nil
}

func (l LineLink) Reject(lock bool) (LineLink, error) {
	if l.Locked {
		return LineLink{}, ErrLineLinkLocked
	}
	return LineLink{
		ItemID:      nil,
		Status:      LinkStatusRejected,
		Method:      LinkMethodManual,
		Locked:      lock,
		Suggestions: emptySuggestions,
	}, nil
}

// RememberedItemID chooses the catalog item id to store in match memory for a decision.
func RememberedItemID(action string, linkedItemID *string, blockedSuggestionID string) *string {
	switch strings.TrimSpace(action) {
	case MemoryActionLink:
		return linkedItemID
	case MemoryActionNeverMatch:
		blocked := strings.TrimSpace(blockedSuggestionID)
		if blocked == "" {
			return nil
		}
		return &blocked
	default:
		return nil
	}
}

func RecalculateLinkingStatus(lineStatuses []string) string {
	if len(lineStatuses) == 0 {
		return LinkingStatusPending
	}
	for _, status := range lineStatuses {
		if status == LinkStatusUnmatched || status == LinkStatusSuggested {
			return LinkingStatusPending
		}
	}
	return LinkingStatusLinked
}

func LineLinkFromRecord(itemID, status, method string, locked bool, suggestions []byte) LineLink {
	link := LineLink{
		Status:      status,
		Method:      method,
		Locked:      locked,
		Suggestions: suggestions,
	}
	if id := strings.TrimSpace(itemID); id != "" {
		link.ItemID = &id
	}
	if len(link.Suggestions) == 0 {
		link.Suggestions = emptySuggestions
	}
	return link
}

func (l LineLink) PersistFields() (itemID *string, status, method string, locked bool, suggestions []byte) {
	suggestions = l.Suggestions
	if len(suggestions) == 0 {
		suggestions = emptySuggestions
	}
	return l.ItemID, l.Status, l.Method, l.Locked, suggestions
}
