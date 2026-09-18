package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const importSelectCols = `id, file_key, file_size_bytes, status, total_rows, created_count, updated_count, failed_count,
	byte_offset, last_file_row, COALESCE(failure_reason, ''),
	requested_by_user_id, requested_by_email, requested_by_name,
	cancelled_by_user_id, cancelled_by_email, cancelled_by_name,
	created_at, updated_at, completed_at, cancelled_at`

func insertItems(ctx context.Context, ex execer, items []domain.Item) error {
	ids := make([]string, len(items))
	names := make([]string, len(items))
	kinds := make([]string, len(items))
	statuses := make([]string, len(items))
	sources := make([]string, len(items))
	codes := make([]*string, len(items))
	created := make([]time.Time, len(items))
	updated := make([]time.Time, len(items))
	for i, item := range items {
		ids[i] = item.ID
		names[i] = item.Name
		kinds[i] = item.Kind
		statuses[i] = item.Status
		sources[i] = item.CreationSource
		codes[i] = nullIfEmpty(item.InternalCode)
		created[i] = item.CreatedAt
		updated[i] = item.UpdatedAt
	}
	_, err := ex.Exec(ctx, `
		INSERT INTO catalog_items (id, name, kind, status, creation_source, internal_code, created_at, updated_at)
		SELECT * FROM UNNEST($1::char(26)[], $2::text[], $3::varchar[], $4::varchar[], $5::varchar[], $6::varchar[], $7::timestamptz[], $8::timestamptz[])
	`, ids, names, kinds, statuses, sources, codes, created, updated)
	if err != nil {
		if isInternalCodeConflict(err) {
			return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
		}
		return fmt.Errorf("insert catalog items: %w", err)
	}
	return nil
}

func updateItems(ctx context.Context, ex execer, items []domain.Item) error {
	ids := make([]string, len(items))
	names := make([]string, len(items))
	kinds := make([]string, len(items))
	statuses := make([]string, len(items))
	codes := make([]*string, len(items))
	updated := make([]time.Time, len(items))
	for i, item := range items {
		ids[i] = item.ID
		names[i] = item.Name
		kinds[i] = item.Kind
		statuses[i] = item.Status
		codes[i] = nullIfEmpty(item.InternalCode)
		updated[i] = item.UpdatedAt
	}
	_, err := ex.Exec(ctx, `
		UPDATE catalog_items AS c SET
			name = u.name,
			kind = u.kind,
			status = u.status,
			internal_code = u.internal_code,
			updated_at = u.updated_at
		FROM UNNEST($1::char(26)[], $2::text[], $3::varchar[], $4::varchar[], $5::varchar[], $6::timestamptz[])
			AS u(id, name, kind, status, internal_code, updated_at)
		WHERE c.id = u.id
	`, ids, names, kinds, statuses, codes, updated)
	if err != nil {
		return fmt.Errorf("update catalog items: %w", err)
	}
	return nil
}

func scanImport(s itemScanner) (domain.CatalogImport, error) {
	var imp domain.CatalogImport
	var cancelledID, cancelledEmail, cancelledName *string
	var completedAt, cancelledAt *time.Time
	err := s.Scan(
		&imp.ID, &imp.FileKey, &imp.FileSizeBytes, &imp.Status, &imp.TotalRows, &imp.CreatedCount, &imp.UpdatedCount, &imp.FailedCount,
		&imp.ByteOffset, &imp.LastFileRow, &imp.FailureReason,
		&imp.RequestedBy.UserID, &imp.RequestedBy.Email, &imp.RequestedBy.Name,
		&cancelledID, &cancelledEmail, &cancelledName,
		&imp.CreatedAt, &imp.UpdatedAt, &completedAt, &cancelledAt,
	)
	if err != nil {
		return domain.CatalogImport{}, err
	}
	if cancelledID != nil && *cancelledID != "" {
		imp.CancelledBy = &domain.ImportActor{UserID: *cancelledID, Email: deref(cancelledEmail), Name: deref(cancelledName)}
	}
	if completedAt != nil {
		imp.CompletedAt = *completedAt
	}
	if cancelledAt != nil {
		imp.CancelledAt = *cancelledAt
	}
	return imp, nil
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (r *CatalogRepository) CreateImport(ctx context.Context, imp domain.CatalogImport) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO catalog_imports (
			id, file_key, file_size_bytes, status, total_rows, created_count, updated_count, failed_count,
			byte_offset, last_file_row, failure_reason,
			requested_by_user_id, requested_by_email, requested_by_name,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$14,$15,$16)
	`, imp.ID, imp.FileKey, imp.FileSizeBytes, imp.Status, imp.TotalRows, imp.CreatedCount, imp.UpdatedCount, imp.FailedCount,
		imp.ByteOffset, imp.LastFileRow, imp.FailureReason,
		imp.RequestedBy.UserID, imp.RequestedBy.Email, imp.RequestedBy.Name,
		imp.CreatedAt, imp.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "an import is already in progress")
		}
		return fmt.Errorf("create catalog import: %w", err)
	}
	return nil
}

func (r *CatalogRepository) UpdateImport(ctx context.Context, imp domain.CatalogImport) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	return updateImport(ctx, pool, imp)
}

func updateImport(ctx context.Context, ex execer, imp domain.CatalogImport) error {
	var cID, cEmail, cName *string
	if imp.CancelledBy != nil {
		cID = &imp.CancelledBy.UserID
		cEmail = &imp.CancelledBy.Email
		cName = &imp.CancelledBy.Name
	}
	var completed, cancelled *time.Time
	if !imp.CompletedAt.IsZero() {
		t := imp.CompletedAt
		completed = &t
	}
	if !imp.CancelledAt.IsZero() {
		t := imp.CancelledAt
		cancelled = &t
	}
	tag, err := ex.Exec(ctx, `
		UPDATE catalog_imports SET
			status=$2, total_rows=$3, created_count=$4, updated_count=$5, failed_count=$6,
			byte_offset=$7, last_file_row=$8, failure_reason=NULLIF($9,''),
			cancelled_by_user_id=$10, cancelled_by_email=$11, cancelled_by_name=$12,
			updated_at=$13, completed_at=$14, cancelled_at=$15
		WHERE id=$1
	`, imp.ID, imp.Status, imp.TotalRows, imp.CreatedCount, imp.UpdatedCount, imp.FailedCount,
		imp.ByteOffset, imp.LastFileRow, imp.FailureReason,
		cID, cEmail, cName, imp.UpdatedAt, completed, cancelled)
	if err != nil {
		return fmt.Errorf("update catalog import: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	return nil
}

func (r *CatalogRepository) GetImportByID(ctx context.Context, id string) (*domain.CatalogImport, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	imp, err := scanImport(pool.QueryRow(ctx, `SELECT `+importSelectCols+` FROM catalog_imports WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog import: %w", err)
	}
	return &imp, nil
}

func (r *CatalogRepository) GetActiveImport(ctx context.Context) (*domain.CatalogImport, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	imp, err := scanImport(pool.QueryRow(ctx, `SELECT `+importSelectCols+` FROM catalog_imports WHERE status IN ('queued','processing') LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active catalog import: %w", err)
	}
	return &imp, nil
}

func (r *CatalogRepository) ListImports(ctx context.Context, filter ports.ImportListFilter) (ports.ImportListPage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return ports.ImportListPage{}, fmt.Errorf("get tenant db pool: %w", err)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	query := `SELECT ` + importSelectCols + ` FROM catalog_imports WHERE 1=1`
	args := []any{}
	n := 1
	if !filter.AfterCreated.IsZero() && filter.AfterID != "" {
		query += fmt.Sprintf(` AND (created_at, id) < ($%d, $%d)`, n, n+1)
		args = append(args, filter.AfterCreated, filter.AfterID)
		n += 2
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC, id DESC LIMIT $%d`, n)
	args = append(args, limit+1)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return ports.ImportListPage{}, fmt.Errorf("list catalog imports: %w", err)
	}
	defer rows.Close()
	out := make([]domain.CatalogImport, 0, limit)
	for rows.Next() {
		imp, err := scanImport(rows)
		if err != nil {
			return ports.ImportListPage{}, err
		}
		out = append(out, imp)
	}
	if err := rows.Err(); err != nil {
		return ports.ImportListPage{}, err
	}
	page := ports.ImportListPage{Items: out}
	if len(out) > limit {
		page.HasMore = true
		page.Items = out[:limit]
	}
	return page, nil
}

func (r *CatalogRepository) InsertImportErrors(ctx context.Context, rows []domain.ImportRowError) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	return insertImportErrors(ctx, pool, rows)
}

func insertImportErrors(ctx context.Context, ex execer, rows []domain.ImportRowError) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]string, len(rows))
	importIDs := make([]string, len(rows))
	fileRows := make([]int32, len(rows))
	cols := make([]*string, len(rows))
	codes := make([]*string, len(rows))
	names := make([]*string, len(rows))
	kinds := make([]*string, len(rows))
	errCodes := make([]string, len(rows))
	msgs := make([]string, len(rows))
	created := make([]time.Time, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		importIDs[i] = row.ImportID
		fileRows[i] = int32(row.FileRow)
		cols[i] = nullIfEmpty(row.Column)
		codes[i] = nullIfEmpty(row.InternalCode)
		names[i] = nullIfEmpty(row.Name)
		kinds[i] = nullIfEmpty(row.Kind)
		errCodes[i] = row.Code
		msgs[i] = row.Message
		created[i] = row.CreatedAt
	}
	_, err := ex.Exec(ctx, `
		INSERT INTO catalog_import_errors (
			id, import_id, file_row, column_name, internal_code, name, kind, code, message, created_at
		)
		SELECT * FROM UNNEST(
			$1::char(26)[], $2::char(26)[], $3::int[], $4::varchar[], $5::text[], $6::text[], $7::text[], $8::varchar[], $9::text[], $10::timestamptz[]
		)
		ON CONFLICT DO NOTHING
	`, ids, importIDs, fileRows, cols, codes, names, kinds, errCodes, msgs, created)
	if err != nil {
		return fmt.Errorf("insert catalog import errors: %w", err)
	}
	return nil
}

func (r *CatalogRepository) ListImportErrors(ctx context.Context, filter ports.ImportErrorListFilter) (ports.ImportErrorListPage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return ports.ImportErrorListPage{}, fmt.Errorf("get tenant db pool: %w", err)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	var total int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM catalog_import_errors WHERE import_id=$1`, filter.ImportID).Scan(&total); err != nil {
		return ports.ImportErrorListPage{}, fmt.Errorf("count catalog import errors: %w", err)
	}
	query := `SELECT id, import_id, file_row, COALESCE(column_name, ''), COALESCE(internal_code, ''), COALESCE(name, ''), COALESCE(kind, ''), code, message, created_at
		FROM catalog_import_errors WHERE import_id=$1`
	args := []any{filter.ImportID}
	n := 2
	if filter.AfterFileRow > 0 && filter.AfterID != "" {
		query += fmt.Sprintf(` AND (file_row, id) > ($%d, $%d)`, n, n+1)
		args = append(args, filter.AfterFileRow, filter.AfterID)
		n += 2
	}
	query += fmt.Sprintf(` ORDER BY file_row ASC, id ASC LIMIT $%d`, n)
	args = append(args, limit+1)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return ports.ImportErrorListPage{}, fmt.Errorf("list catalog import errors: %w", err)
	}
	defer rows.Close()
	out := make([]domain.ImportRowError, 0, limit)
	for rows.Next() {
		var row domain.ImportRowError
		if err := rows.Scan(&row.ID, &row.ImportID, &row.FileRow, &row.Column, &row.InternalCode, &row.Name, &row.Kind, &row.Code, &row.Message, &row.CreatedAt); err != nil {
			return ports.ImportErrorListPage{}, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return ports.ImportErrorListPage{}, err
	}
	page := ports.ImportErrorListPage{Items: out, Total: total}
	if len(out) > limit {
		page.HasMore = true
		page.Items = out[:limit]
	}
	return page, nil
}

func (r *CatalogRepository) PurgeStaleImports(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return 0, fmt.Errorf("get tenant db pool: %w", err)
	}
	var deleted int64
	for {
		tag, err := pool.Exec(ctx, `
			DELETE FROM catalog_imports
			WHERE id IN (
				SELECT id FROM catalog_imports
				WHERE created_at < $1 AND status NOT IN ('queued', 'processing')
				LIMIT $2
			)
		`, before, batchSize)
		if err != nil {
			return deleted, fmt.Errorf("purge catalog imports: %w", err)
		}
		n := tag.RowsAffected()
		deleted += n
		if n < int64(batchSize) {
			break
		}
	}
	return deleted, nil
}

func (r *CatalogRepository) ApplyImportChunk(ctx context.Context, imp domain.CatalogImport, creates, updates []domain.Item, errs []domain.ImportRowError) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import chunk tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanImport(tx.QueryRow(ctx, `SELECT `+importSelectCols+` FROM catalog_imports WHERE id=$1 FOR UPDATE`, imp.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	if err != nil {
		return fmt.Errorf("lock catalog import: %w", err)
	}
	if current.IsTerminal() {
		return tx.Commit(ctx)
	}
	if len(creates) > 0 {
		if err := insertItems(ctx, tx, creates); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		if err := updateItems(ctx, tx, updates); err != nil {
			return err
		}
	}
	if err := insertImportErrors(ctx, tx, errs); err != nil {
		return err
	}
	if err := updateImport(ctx, tx, imp); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import chunk: %w", err)
	}
	return nil
}
