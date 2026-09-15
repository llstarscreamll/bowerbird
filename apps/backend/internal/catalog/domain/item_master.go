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

// ParseImportKind accepts API codes and Spanish CSV aliases. Empty defaults to unknown.
func ParseImportKind(raw string) (ItemKind, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "", KindUnknown, "desconocido":
		return ParseItemKind(KindUnknown)
	case KindGoods, "bien":
		return ParseItemKind(KindGoods)
	case KindService, "servicio":
		return ParseItemKind(KindService)
	case KindAsset, "activo":
		return ParseItemKind(KindAsset)
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

// Assigned reports a present code. The zero value means “not provided”.
func (c InternalCode) Assigned() bool { return c.value != "" }

func (i Item) IsConfirmed() bool {
	return i.Status == StatusConfirmed
}

// MergeInto retires this item into survivorID. Internal code is cleared so the
// unique index can stay on the surviving master.
func (i *Item) MergeInto(survivorID string, now time.Time) error {
	survivorID = strings.TrimSpace(survivorID)
	if survivorID == "" {
		return ErrItemIDRequired
	}
	if i.ID == survivorID {
		return ErrCannotMergeIntoSelf
	}
	if i.Status == StatusMerged {
		if i.MergedIntoID == survivorID {
			return nil
		}
		return ErrItemAlreadyMerged
	}
	i.Status = StatusMerged
	i.MergedIntoID = survivorID
	i.InternalCode = ""
	i.UpdatedAt = now.UTC()
	return nil
}

// ItemKind returns the Kind field as a validated value object.
// Kind is stored as string for persistence; mutate only via New* / ChangeKind.
func (i Item) ItemKind() (ItemKind, error) {
	return ParseItemKind(i.Kind)
}

// ParsedInternalCode returns the assigned internal code, if any.
func (i Item) ParsedInternalCode() (InternalCode, bool) {
	if strings.TrimSpace(i.InternalCode) == "" {
		return InternalCode{}, false
	}
	return InternalCode{value: i.InternalCode}, true
}

func newConfirmedItem(id, name string, kind ItemKind, code InternalCode, source string, now time.Time) (Item, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Item{}, ErrMissingItemName
	}
	if strings.TrimSpace(id) == "" {
		return Item{}, ErrItemIDRequired
	}
	if !code.Assigned() {
		return Item{}, ErrMissingInternalCode
	}
	now = now.UTC()
	return Item{
		ID:             id,
		Name:           name,
		Kind:           kind.String(),
		Status:         StatusConfirmed,
		CreationSource: source,
		InternalCode:   code.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// NewManualItem creates a user-confirmed catalog item with a required internal code.
func NewManualItem(id, name string, kind ItemKind, code InternalCode, now time.Time) (Item, error) {
	return newConfirmedItem(id, name, kind, code, CreationSourceManual, now)
}

// NewImportedItem creates a confirmed catalog item born from a CSV import.
func NewImportedItem(id, name string, kind ItemKind, code InternalCode, now time.Time) (Item, error) {
	return newConfirmedItem(id, name, kind, code, CreationSourceImport, now)
}

func (i *Item) Rename(name string, now time.Time) error {
	if i.IsMerged() {
		return ErrMergedItemFrozen
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrMissingItemName
	}
	i.Name = name
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Item) ChangeKind(kind ItemKind, now time.Time) error {
	if i.IsMerged() {
		return ErrMergedItemFrozen
	}
	i.Kind = kind.String()
	i.UpdatedAt = now.UTC()
	return nil
}

// ApplyImport syncs name/kind from a catalog file and confirms provisionals.
// Does not change InternalCode or CreationSource.
func (i *Item) ApplyImport(name string, kind ItemKind, now time.Time) (bool, error) {
	changed := false
	if strings.TrimSpace(name) != i.Name {
		if err := i.Rename(name, now); err != nil {
			return false, err
		}
		changed = true
	}
	if i.Kind != kind.String() {
		if err := i.ChangeKind(kind, now); err != nil {
			return false, err
		}
		changed = true
	}
	if i.IsProvisional() {
		code, ok := i.ParsedInternalCode()
		if !ok {
			return false, ErrConfirmRequiresCode
		}
		if err := i.Confirm(code, now); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
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
// exist or be supplied (zero InternalCode = not provided in this operation).
func (i *Item) Confirm(provided InternalCode, now time.Time) error {
	if i.IsMerged() {
		return ErrMergedItemFrozen
	}
	if i.IsConfirmed() {
		return ErrItemAlreadyConfirmed
	}
	if !i.IsProvisional() {
		return ErrConfirmRequiresCode
	}
	current, ok := i.ParsedInternalCode()
	switch {
	case ok:
		if provided.Assigned() && !provided.Equals(current) {
			return ErrInternalCodeImmutable
		}
	case provided.Assigned():
		if err := i.AssignInternalCode(provided, now); err != nil {
			return err
		}
	default:
		return ErrConfirmRequiresCode
	}
	i.Status = StatusConfirmed
	i.UpdatedAt = now.UTC()
	return nil
}

// AssignInternalCode allows first assignment only; rejects changes when already set.
func (i *Item) AssignInternalCode(next InternalCode, now time.Time) error {
	if i.IsMerged() {
		return ErrMergedItemFrozen
	}
	if !next.Assigned() {
		return ErrMissingInternalCode
	}
	if current, ok := i.ParsedInternalCode(); ok {
		if !current.Equals(next) {
			return ErrInternalCodeImmutable
		}
		return nil
	}
	i.InternalCode = next.String()
	i.UpdatedAt = now.UTC()
	return nil
}
