package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidItemKind           = errors.New("invalid catalog item kind")
	ErrMissingInternalSKU        = errors.New("missing internal sku")
	ErrInternalSKUImmutable      = errors.New("internal sku cannot be changed once set")
	ErrItemAlreadyConfirmed      = errors.New("item is already confirmed")
	ErrConfirmRequiresSKU        = errors.New("confirming a provisional item requires an internal sku")
	ErrCannotRevertToProvisional = errors.New("cannot revert a confirmed item to provisional")
	ErrInvalidItemStatus         = errors.New("invalid item status")
)

// ItemKind is a validated catalog item classification.
type ItemKind struct {
	value string
}

func ParseItemKind(raw string) (ItemKind, error) {
	v := strings.TrimSpace(raw)
	switch v {
	case KindGoods, KindService, KindAsset, KindUnknown:
		return ItemKind{value: v}, nil
	default:
		return ItemKind{}, ErrInvalidItemKind
	}
}

func (k ItemKind) String() string { return k.value }

func (k ItemKind) Equals(other ItemKind) bool { return k.value == other.value }

// InternalSKU is the tenant-canonical item code (immutable once assigned).
//
// Persistence ACL: InternalSKU is not a column on catalog_items. It is stored as
// an Alias with Scheme=internal_sku (party unset). The Item aggregate enforces
// assignment/immutability rules; the application loads current SKU via
// AliasRepository and persists new aliases via CatalogWriteRepository in the
// same transaction as the Item.
type InternalSKU struct {
	value string
}

func ParseInternalSKU(raw string) (InternalSKU, error) {
	v := NormalizeItemCode(raw)
	if v == "" {
		return InternalSKU{}, ErrMissingInternalSKU
	}
	return InternalSKU{value: v}, nil
}

func (s InternalSKU) String() string { return s.value }

func (s InternalSKU) Equals(other InternalSKU) bool { return s.value == other.value }

func (i Item) IsConfirmed() bool {
	return i.Status == StatusConfirmed
}

// ItemKind returns the Kind field as a validated value object.
// Kind is stored as string for persistence; mutate only via New* / ChangeKind.
func (i Item) ItemKind() (ItemKind, error) {
	return ParseItemKind(i.Kind)
}

// NewManualItem creates a user-confirmed catalog item.
// The required InternalSKU must be persisted separately as an internal_sku Alias
// in the same unit of work (see CatalogWriteRepository).
func NewManualItem(id, name string, kind ItemKind, sku InternalSKU, now time.Time) (Item, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Item{}, ErrMissingItemName
	}
	if strings.TrimSpace(id) == "" {
		return Item{}, ErrItemIDRequired
	}
	if sku.value == "" {
		return Item{}, ErrMissingInternalSKU
	}
	now = now.UTC()
	return Item{
		ID:        id,
		Name:      name,
		Kind:      kind.String(),
		Status:    StatusConfirmed,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewInternalSKUAlias creates an unscoped internal_sku alias for an item.
func NewInternalSKUAlias(id, itemID string, sku InternalSKU, now time.Time) (Alias, error) {
	if sku.value == "" {
		return Alias{}, ErrMissingAliasValue
	}
	if strings.TrimSpace(itemID) == "" {
		return Alias{}, ErrItemIDRequired
	}
	if strings.TrimSpace(id) == "" {
		return Alias{}, ErrItemIDRequired
	}
	now = now.UTC()
	return Alias{
		ID:        id,
		ItemID:    itemID,
		Scheme:    AliasSchemeInternalSKU,
		PartyID:   nil,
		Value:     sku.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (i *Item) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrMissingItemName
	}
	i.Name = name
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Item) ChangeKind(kind ItemKind, now time.Time) {
	i.Kind = kind.String()
	i.UpdatedAt = now.UTC()
}

// InterpretMasterStatusChange validates a master-update status intent.
// confirm=true means the caller must invoke Confirm (with SKU rules).
// Requesting provisional is a no-op when already provisional; it errors only
// when attempting to revert a confirmed item.
func (i Item) InterpretMasterStatusChange(requested string) (confirm bool, err error) {
	requested = strings.TrimSpace(requested)
	switch requested {
	case "":
		return false, nil
	case StatusConfirmed:
		return true, nil
	case StatusProvisional:
		if i.IsConfirmed() {
			return false, ErrCannotRevertToProvisional
		}
		return false, nil
	default:
		return false, ErrInvalidItemStatus
	}
}

// Confirm transitions provisional → confirmed.
// currentSKU / newSKU come from the Alias ACL (not Item fields): pass the
// currently persisted internal_sku (if any) and an optional first assignment.
func (i *Item) Confirm(currentSKU *InternalSKU, newSKU *InternalSKU, now time.Time) (sku InternalSKU, assignNew bool, err error) {
	if i.IsConfirmed() {
		return InternalSKU{}, false, ErrItemAlreadyConfirmed
	}
	if !i.IsProvisional() {
		return InternalSKU{}, false, ErrConfirmRequiresSKU
	}
	switch {
	case currentSKU != nil && currentSKU.value != "":
		sku = *currentSKU
		assignNew = false
	case newSKU != nil && newSKU.value != "":
		sku = *newSKU
		assignNew = true
	default:
		return InternalSKU{}, false, ErrConfirmRequiresSKU
	}
	i.Status = StatusConfirmed
	i.UpdatedAt = now.UTC()
	return sku, assignNew, nil
}

// AssignInternalSKU allows first assignment only; rejects changes when already set.
// current must be the SKU loaded from the Alias ACL (nil/empty = not yet assigned).
func (i *Item) AssignInternalSKU(current *InternalSKU, next InternalSKU, now time.Time) (assignNew bool, err error) {
	if next.value == "" {
		return false, ErrMissingInternalSKU
	}
	if current != nil && current.value != "" {
		if !current.Equals(next) {
			return false, ErrInternalSKUImmutable
		}
		return false, nil
	}
	i.UpdatedAt = now.UTC()
	return true, nil
}
