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
	ErrMissingAliasParty   = errors.New("supplier sku alias requires a party")
	ErrInvalidAliasSource  = errors.New("invalid alias source")
	ErrInvalidAliasScheme  = errors.New("invalid alias scheme")
	ErrInvalidGTIN         = errors.New("invalid gtin")
	ErrGTINMustBeUnscoped  = errors.New("gtin alias must not have a party")
	ErrCannotMergeIntoSelf = errors.New("cannot merge an item into itself")
	ErrItemAlreadyMerged   = errors.New("catalog item was already merged")
	ErrMergedItemFrozen    = errors.New("merged catalog item cannot be modified")
)

const (
	KindGoods   = "goods"
	KindService = "service"
	KindAsset   = "asset"
	KindUnknown = "unknown"

	StatusProvisional = "provisional"
	StatusConfirmed   = "confirmed"
	StatusMerged      = "merged"

	CreationSourceManual  = "manual"
	CreationSourceInvoice = "invoice"
	CreationSourceImport  = "import"

	AliasSchemeSupplierSKU = "supplier_sku"
	AliasSchemeGTIN        = "gtin"

	AliasSourceInvoice = "invoice"
	AliasSourceManual  = "manual"

	SuggestionReasonHardConflict = "hard_conflict"

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
//
// Persistence notes:
//   - Kind/Status are strings for storage; mutate only via domain factories/methods
//     (NewManualItem, NewProvisionalItem, ChangeKind, Confirm, …). Use ItemKind() to
//     read Kind as a value object.
//   - InternalCode is the tenant-canonical item code (empty until assigned). It is
//     not an Alias; supplier identifiers live on Alias (scheme=supplier_sku).
type Item struct {
	ID             string
	Name           string
	Kind           string // persistence; prefer ItemKind() / ChangeKind in domain logic
	Status         string
	CreationSource string
	InternalCode   string
	MergedIntoID   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
		ID:             id,
		Name:           name,
		Kind:           KindUnknown,
		Status:         StatusProvisional,
		CreationSource: CreationSourceInvoice,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (i Item) IsProvisional() bool {
	return i.Status == StatusProvisional
}

func (i Item) IsMerged() bool {
	return i.Status == StatusMerged
}

// Alias maps an external code (e.g. supplier SKU or GTIN) onto an Item.
type Alias struct {
	ID        string
	ItemID    string
	Scheme    string
	PartyID   *string
	Value     string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ParseAliasSource(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	switch v {
	case AliasSourceInvoice, AliasSourceManual:
		return v, nil
	default:
		return "", ErrInvalidAliasSource
	}
}

// NewSupplierSKUAlias creates a hard supplier_sku alias scoped to a party.
func NewSupplierSKUAlias(id, itemID, partyID, code, source string, now time.Time) (Alias, error) {
	value := NormalizeItemCode(code)
	if value == "" {
		return Alias{}, ErrMissingAliasValue
	}
	if strings.TrimSpace(itemID) == "" {
		return Alias{}, ErrItemIDRequired
	}
	party := strings.TrimSpace(partyID)
	if party == "" {
		return Alias{}, ErrMissingAliasParty
	}
	src, err := ParseAliasSource(source)
	if err != nil {
		return Alias{}, err
	}
	now = now.UTC()
	return Alias{
		ID:        id,
		ItemID:    itemID,
		Scheme:    AliasSchemeSupplierSKU,
		PartyID:   &party,
		Value:     value,
		Source:    src,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewGTINAlias creates an unscoped gtin alias.
func NewGTINAlias(id, itemID, raw, source string, now time.Time) (Alias, error) {
	gtin, err := ParseGTIN(raw)
	if err != nil {
		return Alias{}, err
	}
	if strings.TrimSpace(itemID) == "" {
		return Alias{}, ErrItemIDRequired
	}
	src, err := ParseAliasSource(source)
	if err != nil {
		return Alias{}, err
	}
	now = now.UTC()
	return Alias{
		ID:        id,
		ItemID:    itemID,
		Scheme:    AliasSchemeGTIN,
		PartyID:   nil,
		Value:     gtin.String(),
		Source:    src,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a Alias) PointsTo(itemID string) bool {
	return a.ItemID == itemID
}

// ReassignTo points this alias at another item. Merge uses this instead of AddItemAlias
// so the unique (scheme, party, value) tuple can move without a user-facing 409.
func (a *Alias) ReassignTo(itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrItemIDRequired
	}
	a.ItemID = itemID
	return nil
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
	BuyerCode   string
	SellerSKU   string
	GTIN        string
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

// CanMintProvisional encodes the mint gate: party + usable seller SKU.
func CanMintProvisional(partyID, sellerSKU string) bool {
	return strings.TrimSpace(partyID) != "" && SellerSKUUsable(sellerSKU)
}

// LinkedByHardConflict is the outcome when hard identity hits disagree.
func LinkedByHardConflict(itemIDs []string) LineResolutionResult {
	suggestions := make([]Suggestion, 0, len(itemIDs))
	for _, id := range itemIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		suggestions = append(suggestions, Suggestion{ItemID: id, Score: 1, Reason: SuggestionReasonHardConflict})
	}
	return LineResolutionResult{
		Status:      LinkStatusSuggested,
		Suggestions: suggestions,
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
