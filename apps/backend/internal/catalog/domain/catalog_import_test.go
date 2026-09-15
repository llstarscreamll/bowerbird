package domain_test

import (
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogImportLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	actor, err := domain.NewImportActor("user-1", "a@b.co", "Ana", "Ruiz")
	require.NoError(t, err)
	assert.Equal(t, "Ana Ruiz", actor.Name)

	imp, err := domain.NewCatalogImport("01IMPORT000000000000000000", "1-day/tenants/t/uploads/catalog/u/f.csv", 100, actor, now)
	require.NoError(t, err)
	assert.Equal(t, domain.ImportStatusQueued, imp.Status)
	assert.True(t, imp.IsActive())

	require.NoError(t, imp.Start(now.Add(time.Second)))
	assert.Equal(t, domain.ImportStatusProcessing, imp.Status)
	require.NoError(t, imp.RecordChunk(10, 2, 1, 12, 512, now.Add(2*time.Second)))
	assert.Equal(t, int64(10), imp.CreatedCount)
	assert.Equal(t, int64(2), imp.UpdatedCount)
	assert.Equal(t, int64(1), imp.FailedCount)
	assert.Equal(t, 12, imp.LastFileRow)
	require.NoError(t, imp.Complete(13, now.Add(3*time.Second)))
	assert.Equal(t, domain.ImportStatusCompleted, imp.Status)
	assert.ErrorIs(t, imp.Cancel(actor, now.Add(4*time.Second)), domain.ErrImportNotCancellable)
}

func TestCatalogImportCancel(t *testing.T) {
	now := time.Now().UTC()
	actor, err := domain.NewImportActor("user-1", "a@b.co", "", "")
	require.NoError(t, err)
	assert.Equal(t, "a@b.co", actor.Name)
	imp, err := domain.NewCatalogImport("01IMPORT000000000000000001", "key.csv", 1, actor, now)
	require.NoError(t, err)
	canceller, err := domain.NewImportActor("user-2", "c@d.co", "Carlos", "Pérez")
	require.NoError(t, err)
	require.NoError(t, imp.Cancel(canceller, now.Add(time.Second)))
	assert.Equal(t, domain.ImportStatusCancelled, imp.Status)
	require.NotNil(t, imp.CancelledBy)
	assert.Equal(t, "user-2", imp.CancelledBy.UserID)
	assert.ErrorIs(t, imp.Cancel(canceller, now), domain.ErrImportNotCancellable)
}

func TestImportRowError(t *testing.T) {
	now := time.Now().UTC()
	row, err := domain.NewImportRowError("01ERR000000000000000000000", "01IMPORT000000000000000000", 2, domain.ImportColumnInternalCode, "", "x", "", domain.ImportErrorMissingInternalCode, domain.ImportErrorMessage(domain.ImportErrorMissingInternalCode, ""), now)
	require.NoError(t, err)
	assert.Equal(t, 2, row.FileRow)
	assert.Equal(t, "Falta el código interno", row.Message)
	_, err = domain.NewImportRowError("01ERR000000000000000000001", "01IMPORT000000000000000000", 0, "", "", "", "", domain.ImportErrorMalformedRow, "x", now)
	assert.ErrorIs(t, err, domain.ErrImportRowRequired)
}

func TestCatalogImportRowInterpret(t *testing.T) {
	code, name, kind, issue := domain.CatalogImportRow{FileRow: 2, InternalCode: "SKU-1", Name: "Tornillo", Kind: "bien"}.Interpret()
	require.Nil(t, issue)
	assert.Equal(t, "SKU-1", code.String())
	assert.Equal(t, "Tornillo", name)
	assert.Equal(t, domain.KindGoods, kind.String())

	_, _, _, issue = domain.CatalogImportRow{FileRow: 3, Name: "x"}.Interpret()
	require.NotNil(t, issue)
	assert.Equal(t, domain.ImportErrorMissingInternalCode, issue.Code)
	assert.Equal(t, domain.ImportColumnInternalCode, issue.Column)

	_, _, _, issue = domain.CatalogImportRow{FileRow: 4, InternalCode: "A", Name: "x", Kind: "xyz"}.Interpret()
	require.NotNil(t, issue)
	assert.Equal(t, domain.ImportErrorInvalidKind, issue.Code)

	_, _, _, issue = domain.CatalogImportRow{FileRow: 5, InternalCode: "A"}.Interpret()
	require.NotNil(t, issue)
	assert.Equal(t, domain.ImportErrorMissingName, issue.Code)
	assert.Equal(t, domain.ImportColumnName, issue.Column)

	_, _, _, issue = domain.CatalogImportRow{FileRow: 6, Malformed: true}.Interpret()
	require.NotNil(t, issue)
	assert.Equal(t, domain.ImportErrorMalformedRow, issue.Code)
}

func TestCatalogImportGuards(t *testing.T) {
	now := time.Now().UTC()
	actor, err := domain.NewImportActor("user-1", "a@b.co", "Ana", "Ruiz")
	require.NoError(t, err)
	_, err = domain.NewCatalogImport("01IMPORT000000000000000002", "key.csv", domain.MaxImportFileBytes+1, actor, now)
	assert.ErrorIs(t, err, domain.ErrImportFileTooLarge)

	imp, err := domain.NewCatalogImport("01IMPORT000000000000000003", "key.csv", 1, actor, now)
	require.NoError(t, err)
	assert.False(t, imp.ExceedsRowLimit(domain.MaxImportRows))
	assert.True(t, imp.ExceedsRowLimit(domain.MaxImportRows+1))
	assert.Equal(t, int64(0), imp.Processed())
	require.NoError(t, imp.RecordChunk(1, 2, 3, 10, 0, now))
	assert.Equal(t, int64(6), imp.Processed())
	assert.Contains(t, domain.ImportTemplateCSV(), domain.ImportColumnInternalCode)
}
