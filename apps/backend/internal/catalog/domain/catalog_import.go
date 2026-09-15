package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	ImportStatusQueued     = "queued"
	ImportStatusProcessing = "processing"
	ImportStatusCompleted  = "completed"
	ImportStatusFailed     = "failed"
	ImportStatusCancelled  = "cancelled"

	ImportErrorMissingInternalCode = "missing_internal_code"
	ImportErrorMissingName         = "missing_name"
	ImportErrorInvalidKind         = "invalid_kind"
	ImportErrorMalformedRow        = "malformed_row"

	ImportColumnInternalCode = "internal_code"
	ImportColumnName         = "name"
	ImportColumnKind         = "kind"

	MaxImportFileBytes = 500 * 1024 * 1024
	MaxImportRows      = 5_000_000
	ImportChunkRows    = 5000
	ImportEvidenceMax  = 255
	ImportUploadModule = "catalog"
)

var (
	ErrImportIDRequired           = errors.New("import id is required")
	ErrImportFileKeyRequired      = errors.New("import file key is required")
	ErrImportActorRequired        = errors.New("import actor is required")
	ErrImportNotCancellable       = errors.New("import cannot be cancelled")
	ErrImportAlreadyTerminal      = errors.New("import is already finished")
	ErrImportRowRequired          = errors.New("import error file row is required")
	ErrImportErrorMessageRequired = errors.New("import error message is required")
	ErrInvalidImportColumn        = errors.New("invalid import error column")
	ErrImportFileTooLarge         = errors.New("import file exceeds 500 MiB")
)

// ImportActor is an immutable snapshot of a user at the moment of an action.
type ImportActor struct {
	UserID string
	Email  string
	Name   string
}

func NewImportActor(userID, email, firstName, lastName string) (ImportActor, error) {
	id := strings.TrimSpace(userID)
	mail := strings.TrimSpace(email)
	if id == "" || mail == "" {
		return ImportActor{}, ErrImportActorRequired
	}
	name := strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
	if name == "" {
		name = mail
	}
	return ImportActor{UserID: id, Email: mail, Name: name}, nil
}

type CatalogImport struct {
	ID            string
	FileKey       string
	FileSizeBytes int64
	Status        string
	TotalRows     int64
	CreatedCount  int64
	UpdatedCount  int64
	FailedCount   int64
	ByteOffset    int64
	LastFileRow   int
	FailureReason string
	RequestedBy   ImportActor
	CancelledBy   *ImportActor
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   time.Time
	CancelledAt   time.Time
}

func NewCatalogImport(id, fileKey string, fileSizeBytes int64, requester ImportActor, now time.Time) (CatalogImport, error) {
	if strings.TrimSpace(id) == "" {
		return CatalogImport{}, ErrImportIDRequired
	}
	if strings.TrimSpace(fileKey) == "" {
		return CatalogImport{}, ErrImportFileKeyRequired
	}
	if strings.TrimSpace(requester.UserID) == "" || strings.TrimSpace(requester.Email) == "" {
		return CatalogImport{}, ErrImportActorRequired
	}
	if fileSizeBytes > MaxImportFileBytes {
		return CatalogImport{}, ErrImportFileTooLarge
	}
	now = now.UTC()
	return CatalogImport{
		ID:            strings.TrimSpace(id),
		FileKey:       strings.TrimSpace(fileKey),
		FileSizeBytes: fileSizeBytes,
		Status:        ImportStatusQueued,
		RequestedBy:   requester,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (i CatalogImport) IsActive() bool {
	return i.Status == ImportStatusQueued || i.Status == ImportStatusProcessing
}

func (i CatalogImport) IsTerminal() bool {
	return i.Status == ImportStatusCompleted || i.Status == ImportStatusFailed || i.Status == ImportStatusCancelled
}

func (i *CatalogImport) Start(now time.Time) error {
	if i.IsTerminal() {
		return ErrImportAlreadyTerminal
	}
	now = now.UTC()
	i.Status = ImportStatusProcessing
	i.UpdatedAt = now
	return nil
}

func (i CatalogImport) ExceedsRowLimit(dataRows int64) bool {
	return dataRows > MaxImportRows
}

func (i CatalogImport) Processed() int64 {
	return i.CreatedCount + i.UpdatedCount + i.FailedCount
}

func ImportTemplateCSV() string {
	return ImportColumnInternalCode + "," + ImportColumnName + "," + ImportColumnKind + "\nSKU-001,Producto de ejemplo,bien\n"
}

func (i *CatalogImport) RecordChunk(created, updated, failed int64, lastFileRow int, byteOffset int64, now time.Time) error {
	if i.Status == ImportStatusCancelled {
		return nil
	}
	if i.IsTerminal() {
		return ErrImportAlreadyTerminal
	}
	now = now.UTC()
	i.Status = ImportStatusProcessing
	i.CreatedCount += created
	i.UpdatedCount += updated
	i.FailedCount += failed
	if lastFileRow > i.LastFileRow {
		i.LastFileRow = lastFileRow
	}
	if byteOffset > i.ByteOffset {
		i.ByteOffset = byteOffset
	}
	i.UpdatedAt = now
	return nil
}

func (i *CatalogImport) Complete(totalRows int64, now time.Time) error {
	if i.Status == ImportStatusCancelled {
		return nil
	}
	if i.Status == ImportStatusCompleted {
		return nil
	}
	if i.IsTerminal() {
		return ErrImportAlreadyTerminal
	}
	now = now.UTC()
	i.Status = ImportStatusCompleted
	if totalRows > 0 {
		i.TotalRows = totalRows
	}
	i.UpdatedAt = now
	i.CompletedAt = now
	return nil
}

func (i *CatalogImport) Fail(reason string, now time.Time) error {
	if i.Status == ImportStatusCancelled {
		return nil
	}
	if i.IsTerminal() {
		return ErrImportAlreadyTerminal
	}
	now = now.UTC()
	i.Status = ImportStatusFailed
	i.FailureReason = strings.TrimSpace(reason)
	i.UpdatedAt = now
	i.CompletedAt = now
	return nil
}

func (i *CatalogImport) Cancel(actor ImportActor, now time.Time) error {
	if !i.IsActive() {
		return ErrImportNotCancellable
	}
	if strings.TrimSpace(actor.UserID) == "" || strings.TrimSpace(actor.Email) == "" {
		return ErrImportActorRequired
	}
	now = now.UTC()
	copyActor := actor
	i.Status = ImportStatusCancelled
	i.CancelledBy = &copyActor
	i.CancelledAt = now
	i.UpdatedAt = now
	i.CompletedAt = now
	return nil
}

type ImportRowError struct {
	ID           string
	ImportID     string
	FileRow      int
	Column       string
	InternalCode string
	Name         string
	Kind         string
	Code         string
	Message      string
	CreatedAt    time.Time
}

func truncateEvidence(raw string) string {
	v := strings.TrimSpace(raw)
	if len(v) <= ImportEvidenceMax {
		return v
	}
	return v[:ImportEvidenceMax]
}

func NewImportRowError(id, importID string, fileRow int, column, internalCode, name, kind, code, message string, now time.Time) (ImportRowError, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(importID) == "" {
		return ImportRowError{}, ErrImportIDRequired
	}
	if fileRow < 1 {
		return ImportRowError{}, ErrImportRowRequired
	}
	column = strings.TrimSpace(column)
	switch column {
	case "", ImportColumnInternalCode, ImportColumnName, ImportColumnKind:
	default:
		return ImportRowError{}, ErrInvalidImportColumn
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return ImportRowError{}, ErrImportErrorMessageRequired
	}
	code = strings.TrimSpace(code)
	if code == "" {
		code = ImportErrorMalformedRow
	}
	return ImportRowError{
		ID:           strings.TrimSpace(id),
		ImportID:     strings.TrimSpace(importID),
		FileRow:      fileRow,
		Column:       column,
		InternalCode: truncateEvidence(internalCode),
		Name:         truncateEvidence(name),
		Kind:         truncateEvidence(kind),
		Code:         code,
		Message:      message,
		CreatedAt:    now.UTC(),
	}, nil
}

func ImportErrorMessage(code, kindRaw string) string {
	switch code {
	case ImportErrorMissingInternalCode:
		return "Falta el código interno"
	case ImportErrorMissingName:
		return "Falta el nombre"
	case ImportErrorInvalidKind:
		return "El tipo '" + strings.TrimSpace(kindRaw) + "' no es válido. Usa bien, servicio, activo o desconocido"
	default:
		return "La fila no se pudo leer"
	}
}
