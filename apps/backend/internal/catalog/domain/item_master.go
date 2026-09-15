package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidItemKind           = errors.New("invalid catalog item kind")
	ErrMissingInternalCode       = errors.New("missing internal code")
	ErrInternalCodeImmutable     = errors.New("internal code cannot be changed once set")
	ErrItemAlreadyConfirmed      = errors.New("item is already confirmed")
	ErrConfirmRequiresCode       = errors.New("confirming a provisional item requires an internal code")
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

// InternalCode is the tenant-canonical item code (immutable once assigned).
type InternalCode struct {
	value string
}

func ParseInternalCode(raw string) (InternalCode, error) {
	v := NormalizeItemCode(raw)
	if v == "" {
		return InternalCode{}, ErrMissingInternalCode
	}
	return InternalCode{value: v}, nil
}

func (c InternalCode) String() string { return c.value }

func (c InternalCode) Equals(other InternalCode) bool { return c.value == other.value }

func (i Item) IsConfirmed() bool {
	return i.Status == StatusConfirmed
}

// ItemKind returns the Kind field as a validated value object.
// Kind is stored as string for persistence; mutate only via New* / ChangeKind.
func (i Item) ItemKind() (ItemKind, error) {
	return ParseItemKind(i.Kind)
}

// NewManualItem creates a user-confirmed catalog item with a required internal code.
func NewManualItem(id, name string, kind ItemKind, code InternalCode, now time.Time) (Item, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Item{}, ErrMissingItemName
	}
	if strings.TrimSpace(id) == "" {
		return Item{}, ErrItemIDRequired
	}
	if code.value == "" {
		return Item{}, ErrMissingInternalCode
	}
	now = now.UTC()
	return Item{
		ID:             id,
		Name:           name,
		Kind:           kind.String(),
		Status:         StatusConfirmed,
		CreationSource: CreationSourceManual,
		InternalCode:   code.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
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
// confirm=true means the caller must invoke Confirm (with internal-code rules).
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

// Confirm transitions provisional → confirmed. An internal code must already
// exist or be supplied in the same operation.
func (i *Item) Confirm(newCode *InternalCode, now time.Time) error {
	if i.IsConfirmed() {
		return ErrItemAlreadyConfirmed
	}
	if !i.IsProvisional() {
		return ErrConfirmRequiresCode
	}
	if i.InternalCode == "" {
		if newCode == nil {
			return ErrConfirmRequiresCode
		}
		if err := i.AssignInternalCode(*newCode, now); err != nil {
			return err
		}
	} else if newCode != nil && newCode.value != "" && newCode.value != i.InternalCode {
		return ErrInternalCodeImmutable
	}
	i.Status = StatusConfirmed
	i.UpdatedAt = now.UTC()
	return nil
}

// AssignInternalCode allows first assignment only; rejects changes when already set.
func (i *Item) AssignInternalCode(next InternalCode, now time.Time) error {
	if next.value == "" {
		return ErrMissingInternalCode
	}
	if i.InternalCode != "" {
		if i.InternalCode != next.value {
			return ErrInternalCodeImmutable
		}
		return nil
	}
	i.InternalCode = next.String()
	i.UpdatedAt = now.UTC()
	return nil
}
