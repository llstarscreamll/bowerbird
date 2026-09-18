package commands

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/atta/internal/invoices/application/ports"
	platformStorage "github.com/atta/internal/platform/storage"
)

var (
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrInvoiceDocumentNotFound = errors.New("invoice document not found")
)

type DownloadInvoiceDocumentResult struct {
	Filename    string
	ContentType string
	Data        []byte
}

type DownloadInvoiceDocumentCommand struct {
	repo      ports.InvoiceQueryRepository
	fileStore platformStorage.FileStore
}

func NewDownloadInvoiceDocumentCommand(repo ports.InvoiceQueryRepository, fileStore platformStorage.FileStore) *DownloadInvoiceDocumentCommand {
	if repo == nil {
		panic("invoice query repository is required")
	}
	if fileStore == nil {
		panic("file store is required")
	}
	return &DownloadInvoiceDocumentCommand{repo: repo, fileStore: fileStore}
}

func (c *DownloadInvoiceDocumentCommand) Execute(ctx context.Context, invoiceID string) (*DownloadInvoiceDocumentResult, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return nil, ErrInvoiceNotFound
	}

	header, _, err := c.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		if err.Error() == "invoice not found" {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	if header == nil || strings.TrimSpace(header.DocumentRefS3Key) == "" {
		if header == nil {
			return nil, ErrInvoiceNotFound
		}
		return nil, ErrInvoiceDocumentNotFound
	}

	data, err := c.fileStore.ReadFile(ctx, platformStorage.ReadFileInput{Path: header.DocumentRefS3Key})
	if err != nil {
		return nil, fmt.Errorf("read invoice document: %w", err)
	}

	filename := path.Base(header.DocumentRefS3Key)
	if filename == "" || filename == "." || filename == "/" {
		filename = "invoice-document"
	}

	return &DownloadInvoiceDocumentResult{
		Filename:    filename,
		ContentType: contentTypeForFilename(filename),
		Data:        data,
	}, nil
}

func contentTypeForFilename(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".zip":
		return "application/zip"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
