package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bowerbird/internal/invoices/adapters/documentunlock"
	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/invoices/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	platformTemp "github.com/bowerbird/internal/platform/temp"
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
	create *CreateInvoiceCommand,
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
	if create == nil {
		panic("create invoice command is required")
	}
	if passwordResolver == nil {
		panic("document password resolver is required")
	}

	return &CreateInvoicesFromFilesCommand{
		fileStore:        fileStore,
		xmlExtractor:     xmlExtractor,
		llmExtractor:     llmExtractor,
		repo:             repo,
		passwordResolver: passwordResolver,
		create:           create,
		logger:           slog.Default(),
	}
}

type tempFiles struct {
	files []string
	dirs  []string
}

func (t *tempFiles) trackFile(path string) {
	if path != "" {
		t.files = append(t.files, path)
	}
}

func (t *tempFiles) trackDir(path string) {
	if path != "" {
		t.dirs = append(t.dirs, path)
	}
}

func (t *tempFiles) cleanup() {
	for _, path := range t.files {
		_ = os.Remove(path)
	}
	for _, dir := range t.dirs {
		_ = os.RemoveAll(dir)
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

	cleanup := &tempFiles{}
	defer cleanup.cleanup()

	downloadedFiles, err := cmd.downloadFiles(ctx, supportedFiles, cleanup)
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
			extractedFiles, usedPasswordIdx, err := extractSupportedFromZIP(file.S3Key, file.LocalPath, passwords, cleanup)
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
		if err := cmd.processDocumentGroups(ctx, input, sortedFiles, passwords, passwordIDs); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *CreateInvoicesFromFilesCommand) loadDocumentPasswords(ctx context.Context) ([]string, []string, error) {
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
	if secretID == "" {
		return
	}
	if err := cmd.passwordResolver.MarkUsed(ctx, secretID); err != nil {
		cmd.logger.Warn("failed to mark document password as used", "secret_id", secretID, "error", err)
	}
}

func (cmd *CreateInvoicesFromFilesCommand) processDocumentGroups(
	ctx context.Context,
	input contractJobs.ExtractInvoicesFromFilesJob,
	files []extractedDocument,
	passwords []string,
	passwordIDs []string,
) error {
	groups := groupDocuments(files)

	xmlCount, pdfCount := 0, 0
	for _, file := range files {
		switch file.Kind {
		case documentKindXML:
			xmlCount++
		case documentKindPDF:
			pdfCount++
		}
	}
	singleXMLAndPDF := xmlCount == 1 && pdfCount == 1
	batchXMLHandled := false

	for _, group := range groups {
		xmlHandled, err := cmd.tryExtractAndPersistFromDocument(ctx, input, group.xml, passwords, passwordIDs)
		if err != nil {
			return err
		}
		if xmlHandled {
			batchXMLHandled = true
			continue
		}
		if group.pdf == nil {
			continue
		}
		if singleXMLAndPDF && batchXMLHandled {
			continue
		}

		_, err = cmd.tryExtractAndPersistFromDocument(ctx, input, group.pdf, passwords, passwordIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

type documentGroup struct {
	xml *extractedDocument
	pdf *extractedDocument
}

func groupDocuments(files []extractedDocument) []documentGroup {
	byKey := make(map[string]*documentGroup)
	order := make([]string, 0)

	for _, file := range files {
		key := documentGroupKey(file.Filename)
		if _, ok := byKey[key]; !ok {
			byKey[key] = &documentGroup{}
			order = append(order, key)
		}
		group := byKey[key]
		switch file.Kind {
		case documentKindXML:
			if group.xml == nil {
				group.xml = &file
			}
		case documentKindPDF:
			if group.pdf == nil {
				group.pdf = &file
			}
		}
	}

	groups := make([]documentGroup, 0, len(order))
	for _, key := range order {
		groups = append(groups, *byKey[key])
	}
	return groups
}

func documentGroupKey(filename string) string {
	base := strings.ToLower(filepath.Base(filename))
	ext := filepath.Ext(base)
	if ext == "" {
		return base
	}
	return strings.TrimSuffix(base, ext)
}

func (cmd *CreateInvoicesFromFilesCommand) tryExtractAndPersistFromDocument(
	ctx context.Context,
	input contractJobs.ExtractInvoicesFromFilesJob,
	document *extractedDocument,
	passwords []string,
	passwordIDs []string,
) (handled bool, err error) {
	if document == nil {
		return false, nil
	}

	invoice, extractionSource, storageKey, err := cmd.extractInvoiceDocument(ctx, *document, passwords, passwordIDs)
	if err != nil {
		cmd.logger.Warn("invoice extraction failed for document", "filename", document.Filename, "kind", document.Kind, "error", err)
		return false, nil
	}
	if invoice == nil {
		return false, nil
	}

	duplicated, err := cmd.repo.ExistsInvoiceByCUFE(ctx, invoice.CUFE)
	if err != nil {
		return false, fmt.Errorf("check invoice by cufe: %w", err)
	}
	if duplicated {
		cmd.logger.Info("invoice extraction skipped by cufe", "cufe", invoice.CUFE)
		return true, nil
	}

	persisted, err := cmd.create.Execute(ctx, CreateInvoiceInput{
		SourceName:       input.SourceName,
		SourceID:         input.SourceID,
		ExtractionSource: extractionSource,
		StorageKey:       storageKey,
		Invoice:          invoice,
	})
	if err != nil {
		return false, fmt.Errorf("persist invoice: %w", err)
	}

	cmd.logger.Info("invoice extracted and persisted", "source", extractionSource, "cufe", invoice.CUFE, "header_id", persisted.HeaderID)
	return true, nil
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
	Filename  string
	S3Key     string
	Kind      documentKind
	LocalPath string
	Data      []byte
}

func (d extractedDocument) readContent() ([]byte, error) {
	if d.LocalPath != "" {
		return os.ReadFile(d.LocalPath)
	}
	return d.Data, nil
}

func (cmd *CreateInvoicesFromFilesCommand) downloadFiles(ctx context.Context, refs []contractJobs.File, cleanup *tempFiles) ([]extractedDocument, error) {
	documents := make([]extractedDocument, 0, len(refs))
	for _, ref := range refs {
		if ref.Path == "" {
			continue
		}

		if isZipFileRef(ref) {
			localPath, err := cmd.downloadToTemp(ctx, ref.Path, cleanup)
			if err != nil {
				return nil, fmt.Errorf("download zip attachment from key %s: %w", ref.Path, err)
			}
			documents = append(documents, extractedDocument{
				Filename:  ref.Filename,
				S3Key:     ref.Path,
				Kind:      documentKindZIP,
				LocalPath: localPath,
			})
			continue
		}

		data, err := cmd.fileStore.ReadFile(ctx, platformStorage.ReadFileInput{Path: ref.Path})
		if err != nil {
			return nil, fmt.Errorf("read attachment from key %s: %w", ref.Path, err)
		}
		kind := detectDocumentKind(ref.Filename, data)
		if kind == documentKindXML || kind == documentKindPDF {
			documents = append(documents, extractedDocument{Filename: ref.Filename, S3Key: ref.Path, Kind: kind, Data: data})
		}
	}

	return documents, nil
}

func (cmd *CreateInvoicesFromFilesCommand) downloadToTemp(ctx context.Context, objectPath string, cleanup *tempFiles) (string, error) {
	file, err := platformTemp.CreateFile("bowerbird-zip-*")
	if err != nil {
		return "", fmt.Errorf("create temp zip file: %w", err)
	}
	path := file.Name()
	_ = file.Close()
	cleanup.trackFile(path)

	if err := cmd.fileStore.DownloadFile(ctx, platformStorage.DownloadFileInput{
		Path:     objectPath,
		DestPath: path,
	}); err != nil {
		return "", err
	}
	return path, nil
}

func isZipFileRef(ref contractJobs.File) bool {
	switch strings.ToLower(ref.MimeType) {
	case "application/zip", "application/x-zip-compressed", "multipart/x-zip":
		return true
	}
	return strings.EqualFold(filepath.Ext(ref.Filename), ".zip")
}

func extractSupportedFromZIP(s3Key, zipPath string, passwords []string, cleanup *tempFiles) ([]extractedDocument, int, error) {
	destDir, err := platformTemp.MkdirDir("bowerbird-zip-members-*")
	if err != nil {
		return nil, -1, fmt.Errorf("create temp extract dir: %w", err)
	}
	cleanup.trackDir(destDir)

	members, usedIdx, err := documentunlock.ExtractZIPMembersToDir(zipPath, destDir, passwords, func(name, memberPath string) bool {
		kind := detectDocumentKindFromFile(filepath.Base(name), memberPath)
		return kind == documentKindXML || kind == documentKindPDF
	})
	if err != nil {
		return nil, -1, err
	}

	files := make([]extractedDocument, 0, len(members))
	for _, member := range members {
		name := filepath.Base(member.Name)
		kind := detectDocumentKindFromFile(name, member.Path)
		files = append(files, extractedDocument{
			Filename:  name,
			S3Key:     s3Key,
			Kind:      kind,
			LocalPath: member.Path,
		})
	}
	return files, usedIdx, nil
}

func detectDocumentKindFromFile(filename, path string) documentKind {
	if kind := detectDocumentKindByExtension(filename); kind != documentKindOther {
		return kind
	}
	head, err := readFilePrefix(path, 512)
	if err != nil {
		return documentKindOther
	}
	return detectDocumentKind(filename, head)
}

func detectDocumentKindByExtension(filename string) documentKind {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".zip":
		return documentKindZIP
	case ".xml":
		return documentKindXML
	case ".pdf":
		return documentKindPDF
	default:
		return documentKindOther
	}
}

func readFilePrefix(path string, size int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, size)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:n], nil
}

func detectDocumentKind(filename string, data []byte) documentKind {
	if kind := detectDocumentKindByExtension(filename); kind != documentKindOther {
		return kind
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
	content, err := document.readContent()
	if err != nil {
		return nil, "", "", fmt.Errorf("read document content: %w", err)
	}

	if document.Kind == documentKindXML {
		invoice, err := cmd.xmlExtractor.ParseInvoiceXML(content)
		if err != nil {
			return nil, "", "", fmt.Errorf("extract invoice from xml: %w", err)
		}

		return invoice, "xml", document.S3Key, nil
	}

	if document.Kind == documentKindPDF {
		pdfData := content
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
