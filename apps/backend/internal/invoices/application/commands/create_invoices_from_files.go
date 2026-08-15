package commands

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bowerbird/internal/invoices/adapters/documentunlock"
	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

type CreateInvoicesFromFilesCommand struct {
	fileStore        platformStorage.FileStore
	xmlExtractor     ports.InvoiceXMLExtractor
	llmExtractor     ports.InvoiceLLMExtractor
	repo             ports.InvoiceRepository
	passwordResolver ports.DocumentPasswordResolver
	create           *CreateInvoiceCommand
	logger           *slog.Logger
}

func NewCreateInvoicesFromFilesCommand(
	fileStore platformStorage.FileStore,
	xmlExtractor ports.InvoiceXMLExtractor,
	llmExtractor ports.InvoiceLLMExtractor,
	repo ports.InvoiceRepository,
	passwordResolver ports.DocumentPasswordResolver,
) *CreateInvoicesFromFilesCommand {
	if fileStore == nil {
		panic("file store is required")
	}
	if xmlExtractor == nil {
		panic("xml extractor is required")
	}
	if llmExtractor == nil {
		panic("llm extractor is required")
	}
	if repo == nil {
		panic("invoice repository is required")
	}

	return &CreateInvoicesFromFilesCommand{
		fileStore:        fileStore,
		xmlExtractor:     xmlExtractor,
		llmExtractor:     llmExtractor,
		repo:             repo,
		passwordResolver: passwordResolver,
		create:           NewCreateInvoiceCommand(repo),
		logger:           slog.Default(),
	}
}

func (cmd *CreateInvoicesFromFilesCommand) Execute(ctx context.Context, input contractJobs.ExtractInvoicesFromFilesJob) error {
	alreadyProcessed, err := cmd.repo.ExistsBySource(ctx, input.SourceName, input.SourceID)
	if err != nil {
		return fmt.Errorf("check invoice by source: %w", err)
	}
	if alreadyProcessed {
		cmd.logger.Info("invoice extraction skipped by source", "source_name", input.SourceName, "source_id", input.SourceID)
		return nil
	}

	supportedMimeTypes := []string{"application/zip", "application/xml", "application/pdf", "text/xml", "text/pdf", "application/x-zip-compressed", "multipart/x-zip"}
	supportedFiles := make([]contractJobs.File, 0)
	for _, file := range input.Files {
		if slices.Contains(supportedMimeTypes, file.MimeType) {
			supportedFiles = append(supportedFiles, file)
		}
	}

	if len(supportedFiles) == 0 {
		return nil
	}

	downloadedFiles, err := cmd.downloadFiles(ctx, supportedFiles)
	if err != nil {
		return err
	}

	if len(downloadedFiles) == 0 {
		return nil
	}

	passwords, passwordIDs, err := cmd.loadDocumentPasswords(ctx)
	if err != nil {
		cmd.logger.Warn("failed to load document passwords", "error", err)
		passwords = nil
		passwordIDs = nil
	}

	for _, file := range downloadedFiles {
		files := make([]extractedDocument, 0)

		if file.Kind == documentKindZIP {
			extractedFiles, usedPasswordIdx, err := extractSupportedFromZIP(file.S3Key, file.Data, passwords)
			if err != nil {
				cmd.logger.Warn("failed to extract files from zip", "filename", file.Filename, "error", err)
				continue
			}
			if usedPasswordIdx >= 0 && usedPasswordIdx < len(passwordIDs) {
				cmd.markPasswordUsed(ctx, passwordIDs[usedPasswordIdx])
			}

			if len(extractedFiles) == 0 {
				cmd.logger.Warn("no supported files found in zip", "filename", file.Filename)
				continue
			}

			files = append(files, extractedFiles...)
		} else {
			files = append(files, file)
		}

		sortedFiles := sortFiles(files)

		for _, xmlOrPdf := range sortedFiles {
			invoice, extractionSource, storageKey, err := cmd.extractInvoiceDocument(ctx, xmlOrPdf, passwords, passwordIDs)
			if err != nil {
				cmd.logger.Warn("invoice extraction failed for document", "filename", xmlOrPdf.Filename, "kind", xmlOrPdf.Kind, "error", err)
				continue
			}
			if invoice == nil {
				continue
			}

			duplicated, err := cmd.repo.ExistsInvoiceByCUFE(ctx, invoice.CUFE)
			if err != nil {
				return fmt.Errorf("check invoice by cufe: %w", err)
			}
			if duplicated {
				cmd.logger.Info("invoice extraction skipped by cufe", "cufe", invoice.CUFE)
				continue
			}

			persisted, err := cmd.create.Execute(ctx, CreateInvoiceInput{
				SourceName:       input.SourceName,
				SourceID:         input.SourceID,
				ExtractionSource: extractionSource,
				StorageKey:       storageKey,
				Invoice:          invoice,
			})
			if err != nil {
				return fmt.Errorf("persist invoice: %w", err)
			}

			cmd.logger.Info("invoice extracted and persisted", "source", extractionSource, "cufe", invoice.CUFE, "header_id", persisted.HeaderID)
		}
	}

	return nil
}

func (cmd *CreateInvoicesFromFilesCommand) loadDocumentPasswords(ctx context.Context) ([]string, []string, error) {
	if cmd.passwordResolver == nil {
		return nil, nil, nil
	}
	candidates, err := cmd.passwordResolver.ResolveCandidates(ctx)
	if err != nil {
		return nil, nil, err
	}
	passwords := make([]string, 0, len(candidates))
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Value == "" {
			continue
		}
		passwords = append(passwords, candidate.Value)
		ids = append(ids, candidate.SecretID)
	}
	return passwords, ids, nil
}

func (cmd *CreateInvoicesFromFilesCommand) markPasswordUsed(ctx context.Context, secretID string) {
	if cmd.passwordResolver == nil || secretID == "" {
		return
	}
	if err := cmd.passwordResolver.MarkUsed(ctx, secretID); err != nil {
		cmd.logger.Warn("failed to mark document password as used", "secret_id", secretID, "error", err)
	}
}

func sortFiles(files []extractedDocument) []extractedDocument {
	ordered := make([]extractedDocument, 0, len(files))

	for _, file := range files {
		if file.Kind == documentKindXML {
			ordered = append(ordered, file)
		}
	}

	for _, file := range files {
		if file.Kind == documentKindPDF {
			ordered = append(ordered, file)
		}
	}

	return ordered
}

type documentKind string

const (
	documentKindZIP   documentKind = "zip"
	documentKindXML   documentKind = "xml"
	documentKindPDF   documentKind = "pdf"
	documentKindOther documentKind = "other"
)

type extractedDocument struct {
	Filename string
	S3Key    string
	Kind     documentKind
	Data     []byte
}

func (cmd *CreateInvoicesFromFilesCommand) downloadFiles(ctx context.Context, refs []contractJobs.File) ([]extractedDocument, error) {
	documents := make([]extractedDocument, 0, len(refs))
	for _, ref := range refs {
		if ref.Path == "" {
			continue
		}

		data, err := cmd.fileStore.ReadFile(ctx, platformStorage.ReadFileInput{Path: ref.Path})
		if err != nil {
			return nil, fmt.Errorf("read attachment from key %s: %w", ref.Path, err)
		}
		kind := detectDocumentKind(ref.Filename, data)
		if kind == documentKindXML || kind == documentKindPDF || kind == documentKindZIP {
			documents = append(documents, extractedDocument{Filename: ref.Filename, S3Key: ref.Path, Kind: kind, Data: data})
			continue
		}
	}

	return documents, nil
}

func extractSupportedFromZIP(s3Key string, data []byte, passwords []string) ([]extractedDocument, int, error) {
	members, usedIdx, err := documentunlock.ExtractZIPMembers(data, passwords, func(name string, content []byte) bool {
		kind := detectDocumentKind(filepath.Base(name), content)
		return kind == documentKindXML || kind == documentKindPDF
	})
	if err != nil {
		return nil, -1, err
	}

	files := make([]extractedDocument, 0, len(members))
	for _, member := range members {
		name := filepath.Base(member.Name)
		kind := detectDocumentKind(name, member.Data)
		files = append(files, extractedDocument{
			Filename: name,
			S3Key:    s3Key,
			Kind:     kind,
			Data:     member.Data,
		})
	}
	return files, usedIdx, nil
}

func detectDocumentKind(filename string, data []byte) documentKind {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return documentKindZIP
	case ".xml":
		return documentKindXML
	case ".pdf":
		return documentKindPDF
	}

	if len(data) >= 4 {
		if data[0] == 'P' && data[1] == 'K' {
			return documentKindZIP
		}
		if bytes.HasPrefix(data, []byte("%PDF")) {
			return documentKindPDF
		}
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		return documentKindXML
	}

	return documentKindOther
}

func (cmd *CreateInvoicesFromFilesCommand) extractInvoiceDocument(
	ctx context.Context,
	document extractedDocument,
	passwords []string,
	passwordIDs []string,
) (*domain.InvoiceDocument, string, string, error) {
	if document.Kind == documentKindXML {
		invoice, err := cmd.xmlExtractor.ParseInvoiceXML(document.Data)
		if err != nil {
			return nil, "", "", fmt.Errorf("extract invoice from xml: %w", err)
		}

		return invoice, "xml", document.S3Key, nil
	}

	if document.Kind == documentKindPDF {
		pdfData := document.Data
		if documentunlock.IsPDFEncrypted(pdfData) {
			unlocked, idx, err := documentunlock.TryUnlockPDF(pdfData, passwords)
			if err != nil {
				return nil, "", "", fmt.Errorf("unlock encrypted pdf: %w", err)
			}
			pdfData = unlocked
			if idx >= 0 && idx < len(passwordIDs) {
				cmd.markPasswordUsed(ctx, passwordIDs[idx])
			}
		}

		invoice, err := cmd.llmExtractor.ExtractFromPDF(ctx, pdfData)
		if err != nil {
			return nil, "", "", fmt.Errorf("extract invoice from pdf with llm: %w", err)
		}

		return invoice, "llm", document.S3Key, nil
	}

	return nil, "", "", nil
}
