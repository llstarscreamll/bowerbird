package v1

import (
	"fmt"
	"path"
	"strings"
)

const queueInvoiceExtractionDataType = "queue-invoice-extraction"

type requestAttributes interface {
	Validate() error
}

type jsonApiRequestDocument[T requestAttributes] struct {
	Data jsonApiDocument[T] `json:"data"`
}

func (r *jsonApiRequestDocument[T]) Validate(expectedDataType string) error {
	if strings.TrimSpace(r.Data.Type) != expectedDataType {
		return fmt.Errorf("data.type must be %s", expectedDataType)
	}

	if err := r.Data.Attributes.Validate(); err != nil {
		return fmt.Errorf("data.attributes: %w", err)
	}

	return nil
}

type queueInvoiceExtractionRequestDocument jsonApiRequestDocument[queueInvoiceExtractionRequestDataAttrs]

func (r *queueInvoiceExtractionRequestDocument) Validate() error {
	return (*jsonApiRequestDocument[queueInvoiceExtractionRequestDataAttrs])(r).Validate(queueInvoiceExtractionDataType)
}

type queueInvoiceExtractionRequestDataAttrs struct {
	Files []file `json:"files"`
}

func (r queueInvoiceExtractionRequestDataAttrs) Validate() error {
	if len(r.Files) == 0 {
		return fmt.Errorf("files is required")
	}

	const maxFiles = 50

	if len(r.Files) > maxFiles {
		return fmt.Errorf("files exceeds max length of %d", maxFiles)
	}

	for idx, file := range r.Files {
		if err := file.Validate(); err != nil {
			return fmt.Errorf("files[%d]: %w", idx, err)
		}
	}

	return nil
}

type file struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
}

func (r file) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("path is required")
	}

	fileType, err := mimeTypeToFileType(r.MimeType)
	if err != nil {
		return err
	}

	if err := validateFileExtensionMatchesType(r.Name, fileType); err != nil {
		return fmt.Errorf("name: %w", err)
	}

	if err := validateFileExtensionMatchesType(r.Path, fileType); err != nil {
		return fmt.Errorf("path: %w", err)
	}

	return nil
}

func mimeTypeToFileType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))

	switch normalized {
	case "application/zip":
		return "zip", nil
	case "application/xml", "text/xml":
		return "xml", nil
	case "application/pdf":
		return "pdf", nil
	default:
		return "", fmt.Errorf("mime_type must be one of: application/zip, application/xml, text/xml, application/pdf")
	}
}

func validateFileExtensionMatchesType(value, expectedType string) error {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(value)), "."))
	if ext == "" {
		return fmt.Errorf("file extension is required")
	}

	if ext != expectedType {
		return fmt.Errorf("file extension must match mime_type %q", expectedType)
	}

	return nil
}
