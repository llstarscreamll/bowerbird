package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidMemoryAction = errors.New("invalid memory action")
	ErrItemIDRequired      = errors.New("item id is required for link")
	ErrMissingItemName     = errors.New("missing catalog item name")
	ErrMissingAliasValue   = errors.New("missing alias value")
)

const (
	KindGoods   = "goods"
	KindService = "service"
	KindAsset   = "asset"
	KindUnknown = "unknown"

	StatusProvisional = "provisional"
	StatusConfirmed   = "confirmed"

	AliasSchemeSupplierSKU = "supplier_sku"
	AliasSchemeInternalSKU = "internal_sku"

	LinkStatusUnmatched = "unmatched"
	LinkStatusSuggested = "suggested"
	LinkStatusLinked    = "linked"
	LinkStatusRejected  = "rejected"

	LinkMethodMemory = "memory"
	LinkMethodHard   = "hard"
	LinkMethodSoft   = "soft"
	LinkMethodManual = "manual"

	MemoryActionLink       = "link"
	MemoryActionNeverMatch = "never_match"

	EvidenceKindCode            = "code"
	EvidenceKindDescription     = "description"
	EvidenceKindCodeDescription = "code+description"
)

// Item is the catalog aggregate root (goods/service/asset identity).
type Item struct {
	ID        string
	Name      string
	Kind      string
	Status    string
	Stockable *bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewProvisionalItem mints a provisional catalog item from invoice evidence.
func NewProvisionalItem(id, description, fallbackCode string, now time.Time) (Item, error) {
	name := strings.TrimSpace(description)
	if name == "" {
		name = strings.TrimSpace(fallbackCode)
	}
	if name == "" {
		return Item{}, ErrMissingItemName
	}
	now = now.UTC()
	return Item{
		ID:        id,
		Name:      name,
		Kind:      KindUnknown,
		Status:    StatusProvisional,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (i Item) IsProvisional() bool {
	return i.Status == StatusProvisional
}

// Alias maps an external code (e.g. supplier SKU) onto an Item.
type Alias struct {
	ID        string
	ItemID    string
	Scheme    string
	PartyID   *string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewSupplierSKUAlias creates a hard supplier_sku alias scoped to a party.
func NewSupplierSKUAlias(id, itemID, partyID, code string, now time.Time) (Alias, error) {
	value := NormalizeItemCode(code)
	if value == "" {
		return Alias{}, ErrMissingAliasValue
	}
	if strings.TrimSpace(itemID) == "" {
		return Alias{}, ErrItemIDRequired
	}
	now = now.UTC()
	party := strings.TrimSpace(partyID)
	var partyPtr *string
	if party != "" {
		partyPtr = &party
	}
	return Alias{
		ID:        id,
		ItemID:    itemID,
		Scheme:    AliasSchemeSupplierSKU,
		PartyID:   partyPtr,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a Alias) PointsTo(itemID string) bool {
	return a.ItemID == itemID
}

// MatchMemory records a durable human/system matching decision for evidence.
type MatchMemory struct {
	ID                     string
	EvidenceKey            string
	PartyID                *string
	ItemCode               string
	DescriptionFingerprint string
	EvidenceKind           string
	ItemID                 *string
	Action                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// NewMatchMemory builds remembered evidence for link or never_match actions.
func NewMatchMemory(
	id, partyID, itemCode, description, action string,
	itemID *string,
	now time.Time,
) (MatchMemory, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		action = MemoryActionLink
	}
	if action != MemoryActionLink && action != MemoryActionNeverMatch {
		return MatchMemory{}, ErrInvalidMemoryAction
	}
	code := NormalizeItemCode(itemCode)
	descFP := DescriptionFingerprint(description)
	kind := InferEvidenceKind(code, description)
	key := EvidenceKey(partyID, code, descFP, kind)
	now = now.UTC()

	var partyPtr *string
	if strings.TrimSpace(partyID) != "" {
		p := strings.TrimSpace(partyID)
		partyPtr = &p
	}

	memItemID := itemID
	if action == MemoryActionNeverMatch && itemID != nil && strings.TrimSpace(*itemID) != "" {
		blocked := strings.TrimSpace(*itemID)
		memItemID = &blocked
	}
	if action == MemoryActionLink && (itemID == nil || strings.TrimSpace(*itemID) == "") {
		return MatchMemory{}, ErrItemIDRequired
	}

	return MatchMemory{
		ID:                     id,
		EvidenceKey:            key,
		PartyID:                partyPtr,
		ItemCode:               code,
		DescriptionFingerprint: descFP,
		EvidenceKind:           kind,
		ItemID:                 memItemID,
		Action:                 action,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, nil
}

func (m MatchMemory) IsNeverMatch() bool {
	return m.Action == MemoryActionNeverMatch
}

func (m MatchMemory) LinkedItemID() string {
	if m.ItemID == nil {
		return ""
	}
	return *m.ItemID
}

type Suggestion struct {
	ItemID string  `json:"item_id"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type LineResolutionInput struct {
	LineID      string
	PartyID     string
	ItemCode    string
	Description string
	// Existing link state (from DB) for lock / reprocess.
	ExistingItemID string
	ExistingLocked bool
	ExistingStatus string
	ExistingMethod string
}

type LineResolutionResult struct {
	ItemID      string
	Status      string
	Method      string
	Suggestions []Suggestion
	Minted      bool
}

// PreserveLockedLink returns the prior link when the line is locked.
func PreserveLockedLink(input LineResolutionInput) *LineResolutionResult {
	if !input.ExistingLocked || strings.TrimSpace(input.ExistingItemID) == "" {
		return nil
	}
	method := strings.TrimSpace(input.ExistingMethod)
	if method == "" {
		method = LinkMethodManual
	}
	return &LineResolutionResult{
		ItemID: input.ExistingItemID,
		Status: LinkStatusLinked,
		Method: method,
	}
}

// LinkedByMemory is the outcome of a positive match-memory hit.
func LinkedByMemory(itemID string) LineResolutionResult {
	return LineResolutionResult{
		ItemID: itemID,
		Status: LinkStatusLinked,
		Method: LinkMethodMemory,
	}
}

// LinkedByHardAlias is the outcome of a supplier_sku hard hit.
func LinkedByHardAlias(itemID string) LineResolutionResult {
	return LineResolutionResult{
		ItemID: itemID,
		Status: LinkStatusLinked,
		Method: LinkMethodHard,
	}
}

// LinkedByProvisionalMint is the outcome after minting a provisional item+alias.
func LinkedByProvisionalMint(itemID string, minted bool, suggestions []Suggestion) LineResolutionResult {
	return LineResolutionResult{
		ItemID:      itemID,
		Status:      LinkStatusLinked,
		Method:      LinkMethodHard,
		Suggestions: suggestions,
		Minted:      minted,
	}
}

// SoftOrUnmatchedStatus chooses suggested vs unmatched from soft matches.
func SoftOrUnmatchedStatus(suggestions []Suggestion) string {
	if len(suggestions) > 0 {
		return LinkStatusSuggested
	}
	return LinkStatusUnmatched
}

// FilterBlockedSuggestions drops soft hits for a never_match blocked item.
func FilterBlockedSuggestions(suggestions []Suggestion, blockedItemID *string) []Suggestion {
	if blockedItemID == nil || strings.TrimSpace(*blockedItemID) == "" {
		return suggestions
	}
	out := make([]Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if s.ItemID == *blockedItemID {
			continue
		}
		out = append(out, s)
	}
	return out
}

// CanMintProvisional encodes the analytics-first mint gate: party + non-empty code.
func CanMintProvisional(partyID, itemCode string) bool {
	return strings.TrimSpace(partyID) != "" && NormalizeItemCode(itemCode) != ""
}

// ManualLinkDecision is the invoice-line outcome of a human review action.
type ManualLinkDecision struct {
	ItemID *string
	Status string
	Method string
}

// DecideManualLink derives line link fields for link | never_match.
func DecideManualLink(action, itemID string) (ManualLinkDecision, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		action = MemoryActionLink
	}
	switch action {
	case MemoryActionLink:
		if strings.TrimSpace(itemID) == "" {
			return ManualLinkDecision{}, ErrItemIDRequired
		}
		idCopy := strings.TrimSpace(itemID)
		return ManualLinkDecision{
			ItemID: &idCopy,
			Status: LinkStatusLinked,
			Method: LinkMethodManual,
		}, nil
	case MemoryActionNeverMatch:
		return ManualLinkDecision{
			ItemID: nil,
			Status: LinkStatusRejected,
			Method: LinkMethodManual,
		}, nil
	default:
		return ManualLinkDecision{}, ErrInvalidMemoryAction
	}
}

func NormalizeItemCode(code string) string {
	return strings.TrimSpace(code)
}

func NormalizeDescription(desc string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(desc))), " ")
}

func DescriptionFingerprint(desc string) string {
	normalized := NormalizeDescription(desc)
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:16])
}

func EvidenceKey(partyID, itemCode, descFingerprint, kind string) string {
	raw := strings.Join([]string{
		strings.TrimSpace(partyID),
		NormalizeItemCode(itemCode),
		descFingerprint,
		kind,
	}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func InferEvidenceKind(itemCode, description string) string {
	hasCode := NormalizeItemCode(itemCode) != ""
	hasDesc := DescriptionFingerprint(description) != ""
	switch {
	case hasCode && hasDesc:
		return EvidenceKindCodeDescription
	case hasCode:
		return EvidenceKindCode
	default:
		return EvidenceKindDescription
	}
}
