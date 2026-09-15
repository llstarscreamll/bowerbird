package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	contractJobs "github.com/bowerbird/internal/catalog/contracts/jobs"
	"github.com/bowerbird/internal/catalog/domain"
	filesapi "github.com/bowerbird/internal/files/api"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
)

type ProcessCatalogImportCommand struct {
	imports   ports.ImportRepository
	items     ports.ItemRepository
	files     filesapi.TenantObjects
	jobQueue  jobs.TaskQueue
	now       func() time.Time
	newID     func() string
	chunkRows int
}

func NewProcessCatalogImportCommand(
	imports ports.ImportRepository,
	items ports.ItemRepository,
	files filesapi.TenantObjects,
	jobQueue jobs.TaskQueue,
) *ProcessCatalogImportCommand {
	if imports == nil {
		panic("import repository is required")
	}
	if items == nil {
		panic("item repository is required")
	}
	if files == nil {
		panic("tenant objects are required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}
	return &ProcessCatalogImportCommand{
		imports:   imports,
		items:     items,
		files:     files,
		jobQueue:  jobQueue,
		now:       time.Now,
		newID:     id.NewULID,
		chunkRows: domain.ImportChunkRows,
	}
}

func (cmd *ProcessCatalogImportCommand) Execute(ctx context.Context, job contractJobs.CatalogImportRequestedJob) error {
	imp, err := cmd.imports.GetImportByID(ctx, job.ImportID)
	if err != nil {
		return err
	}
	if imp == nil {
		return appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	if !imp.IsActive() {
		return nil
	}
	now := cmd.now().UTC()
	if imp.Status == domain.ImportStatusQueued {
		if err := imp.Start(now); err != nil {
			return err
		}
		if err := cmd.imports.UpdateImport(ctx, *imp); err != nil {
			return err
		}
	}

	comma, idx, err := cmd.readImportHeader(ctx, imp.FileKey)
	if err != nil {
		return cmd.fail(ctx, imp, err.Error(), now)
	}
	opened, err := cmd.files.Open(ctx, domain.ImportUploadModule, imp.FileKey, imp.ByteOffset)
	if err != nil {
		return cmd.fail(ctx, imp, "no se pudo leer el archivo de importación", now)
	}
	defer opened.Body.Close()

	chunk, err := cmd.readChunk(opened.Body, *imp, comma, idx)
	if err != nil {
		return cmd.fail(ctx, imp, err.Error(), now)
	}

	creates, updates, rowErrs, err := cmd.buildUpserts(ctx, imp.ID, chunk.rows, now)
	if err != nil {
		return err
	}
	if err := imp.RecordChunk(int64(len(creates)), int64(len(updates)), int64(len(rowErrs)), chunk.lastFileRow, chunk.byteOffset, now); err != nil {
		return err
	}
	if err := cmd.imports.ApplyImportChunk(ctx, *imp, creates, updates, rowErrs); err != nil {
		return err
	}

	fresh, err := cmd.imports.GetImportByID(ctx, imp.ID)
	if err != nil {
		return err
	}
	if fresh == nil || !fresh.IsActive() {
		return nil
	}
	totalRows := int64(chunk.lastFileRow - 1)
	if fresh.ExceedsRowLimit(totalRows) {
		return cmd.fail(ctx, fresh, "el archivo supera el límite de 5.000.000 de filas", now)
	}
	if !chunk.done {
		return cmd.enqueueNext(ctx, fresh.ID)
	}
	if err := fresh.Complete(totalRows, now); err != nil {
		return err
	}
	return cmd.imports.UpdateImport(ctx, *fresh)
}

func (cmd *ProcessCatalogImportCommand) fail(ctx context.Context, imp *domain.CatalogImport, reason string, now time.Time) error {
	if err := imp.Fail(reason, now); err != nil {
		return err
	}
	return cmd.imports.UpdateImport(ctx, *imp)
}

func (cmd *ProcessCatalogImportCommand) enqueueNext(ctx context.Context, importID string) error {
	payload, err := contractJobs.MarshalCatalogImportRequested(contractJobs.CatalogImportRequestedJob{ImportID: importID})
	if err != nil {
		return err
	}
	return cmd.jobQueue.Enqueue(ctx, jobs.Job{Type: contractJobs.CatalogImportRequestedType, Payload: payload})
}

type parsedRow struct {
	fileRow      int
	internalCode string
	name         string
	kind         string
	malformed    bool
}

type chunkResult struct {
	rows        []parsedRow
	lastFileRow int
	byteOffset  int64
	done        bool
}

type headerIndex struct {
	code int
	name int
	kind int
}

func (cmd *ProcessCatalogImportCommand) readImportHeader(ctx context.Context, fileKey string) (rune, headerIndex, error) {
	opened, err := cmd.files.Open(ctx, domain.ImportUploadModule, fileKey, 0)
	if err != nil {
		return 0, headerIndex{}, errors.New("no se pudo leer el archivo de importación")
	}
	defer opened.Body.Close()
	br := bufio.NewReader(opened.Body)
	if _, err := skipBOM(br); err != nil {
		if errors.Is(err, io.EOF) {
			return 0, headerIndex{}, errors.New("el archivo está vacío")
		}
		return 0, headerIndex{}, errors.New("el archivo no se pudo leer")
	}
	raw, err := readCSVRecord(br)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, headerIndex{}, errors.New("el archivo está vacío")
		}
		return 0, headerIndex{}, errors.New("el encabezado no es válido. Se esperan las columnas internal_code y name")
	}
	comma := detectDelimiter(string(raw))
	record, err := parseCSVRecord(raw, comma)
	if err != nil {
		return 0, headerIndex{}, errors.New("el encabezado no es válido. Se esperan las columnas internal_code y name")
	}
	idx, err := mapImportHeader(record)
	if err != nil {
		return 0, headerIndex{}, err
	}
	return comma, idx, nil
}

func (cmd *ProcessCatalogImportCommand) readChunk(body io.Reader, imp domain.CatalogImport, comma rune, idx headerIndex) (chunkResult, error) {
	br := bufio.NewReader(body)
	out := chunkResult{lastFileRow: 1, byteOffset: imp.ByteOffset}
	if imp.ByteOffset == 0 {
		bom, err := skipBOM(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return chunkResult{}, errors.New("el archivo está vacío")
			}
			return chunkResult{}, errors.New("el archivo no se pudo leer")
		}
		out.byteOffset += bom
		headerRaw, err := readCSVRecord(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return chunkResult{}, errors.New("el archivo está vacío")
			}
			return chunkResult{}, errors.New("el encabezado no es válido. Se esperan las columnas internal_code y name")
		}
		out.byteOffset += int64(len(headerRaw))
	} else {
		out.lastFileRow = imp.LastFileRow
		if out.lastFileRow < 1 {
			out.lastFileRow = 1
		}
	}

	limit := cmd.chunkRows
	if limit <= 0 {
		limit = domain.ImportChunkRows
	}
	fileRow := out.lastFileRow
	for {
		raw, err := readCSVRecord(br)
		if errors.Is(err, io.EOF) {
			out.done = true
			break
		}
		if err != nil {
			return chunkResult{}, errors.New("el archivo no se pudo leer")
		}
		out.byteOffset += int64(len(raw))
		fileRow++
		out.lastFileRow = fileRow
		if imp.ExceedsRowLimit(int64(fileRow - 1)) {
			return chunkResult{}, errors.New("el archivo supera el límite de 5.000.000 de filas")
		}
		record, parseErr := parseCSVRecord(raw, comma)
		if parseErr != nil {
			out.rows = append(out.rows, parsedRow{fileRow: fileRow, malformed: true})
		} else {
			out.rows = append(out.rows, parsedRow{
				fileRow:      fileRow,
				internalCode: cell(record, idx.code),
				name:         cell(record, idx.name),
				kind:         cell(record, idx.kind),
			})
		}
		if len(out.rows) >= limit {
			_, peekErr := br.Peek(1)
			out.done = errors.Is(peekErr, io.EOF)
			break
		}
	}
	return out, nil
}

func mapImportHeader(header []string) (headerIndex, error) {
	idx := headerIndex{code: -1, name: -1, kind: -1}
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(col, "\ufeff"))) {
		case domain.ImportColumnInternalCode:
			idx.code = i
		case domain.ImportColumnName:
			idx.name = i
		case domain.ImportColumnKind:
			idx.kind = i
		}
	}
	if idx.code < 0 || idx.name < 0 {
		return headerIndex{}, errors.New("el encabezado no es válido. Se esperan las columnas internal_code y name")
	}
	return idx, nil
}

func cell(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}

func detectDelimiter(headerLine string) rune {
	line := strings.TrimRight(headerLine, "\r\n")
	if strings.Count(line, ";") > strings.Count(line, ",") {
		return ';'
	}
	return ','
}

func skipBOM(br *bufio.Reader) (int64, error) {
	bom, err := br.Peek(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if len(bom) >= 3 && bytes.Equal(bom[:3], []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
		return 3, nil
	}
	if len(bom) == 0 {
		return 0, io.EOF
	}
	return 0, nil
}

func readCSVRecord(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	inQuotes := false
	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buf.Len() == 0 {
					return nil, io.EOF
				}
				return buf.Bytes(), nil
			}
			return nil, err
		}
		buf.WriteByte(b)
		if b == '"' {
			if inQuotes {
				next, peekErr := r.Peek(1)
				if peekErr == nil && next[0] == '"' {
					escaped, _ := r.ReadByte()
					buf.WriteByte(escaped)
					continue
				}
			}
			inQuotes = !inQuotes
		}
		if !inQuotes && b == '\n' {
			return buf.Bytes(), nil
		}
	}
}

func parseCSVRecord(raw []byte, comma rune) ([]string, error) {
	line := bytes.TrimRight(raw, "\r\n")
	reader := csv.NewReader(bytes.NewReader(line))
	reader.Comma = comma
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	return reader.Read()
}

func (cmd *ProcessCatalogImportCommand) buildUpserts(ctx context.Context, importID string, rows []parsedRow, now time.Time) ([]domain.Item, []domain.Item, []domain.ImportRowError, error) {
	type accepted struct {
		code domain.InternalCode
		name string
		kind domain.ItemKind
	}
	pending := make(map[string]accepted, len(rows))
	order := make([]string, 0, len(rows))
	errs := make([]domain.ImportRowError, 0)
	for _, row := range rows {
		code, name, kind, issue := domain.CatalogImportRow{
			FileRow:      row.fileRow,
			InternalCode: row.internalCode,
			Name:         row.name,
			Kind:         row.kind,
			Malformed:    row.malformed,
		}.Interpret()
		if issue != nil {
			rowErr, err := domain.NewImportRowError(cmd.newID(), importID, row.fileRow, issue.Column, row.internalCode, row.name, row.kind, issue.Code, issue.Message, now)
			if err != nil {
				return nil, nil, nil, err
			}
			errs = append(errs, rowErr)
			continue
		}
		key := code.String()
		if _, ok := pending[key]; !ok {
			order = append(order, key)
		}
		pending[key] = accepted{code: code, name: name, kind: kind}
	}

	codes := make([]string, 0, len(pending))
	for _, key := range order {
		codes = append(codes, key)
	}
	existing, err := cmd.items.GetItemsByInternalCodes(ctx, codes)
	if err != nil {
		return nil, nil, nil, err
	}
	byCode := make(map[string]domain.Item, len(existing))
	for _, item := range existing {
		byCode[item.InternalCode] = item
	}

	creates := make([]domain.Item, 0)
	updates := make([]domain.Item, 0)
	for _, key := range order {
		row := pending[key]
		if current, ok := byCode[key]; ok {
			changed, err := current.ApplyImport(row.name, row.kind, now)
			if err != nil {
				return nil, nil, nil, err
			}
			if changed {
				updates = append(updates, current)
			}
			continue
		}
		item, err := domain.NewImportedItem(cmd.newID(), row.name, row.kind, row.code, now)
		if err != nil {
			return nil, nil, nil, err
		}
		creates = append(creates, item)
	}
	return creates, updates, errs, nil
}
