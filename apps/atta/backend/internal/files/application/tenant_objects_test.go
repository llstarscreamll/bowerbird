package application_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/atta/internal/files/application"
	platformStorage "github.com/atta/internal/platform/storage"
	"github.com/atta/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	data map[string][]byte
}

func (m *memStore) WriteFileIfAbsent(context.Context, platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}
func (m *memStore) ReadFile(context.Context, platformStorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}
func (m *memStore) OpenFile(_ context.Context, input platformStorage.OpenFileInput) (*platformStorage.OpenFileResult, error) {
	data, ok := m.data[input.Path]
	if !ok {
		return nil, io.EOF
	}
	if input.Offset > int64(len(data)) {
		input.Offset = int64(len(data))
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	return &platformStorage.OpenFileResult{Body: io.NopCloser(bytes.NewReader(data[input.Offset:])), SizeBytes: int64(len(data))}, nil
}
func (m *memStore) DownloadFile(context.Context, platformStorage.DownloadFileInput) error {
	return nil
}
func (m *memStore) Exists(context.Context, platformStorage.ExistsFileInput) (bool, error) {
	return false, nil
}
func (m *memStore) MoveFile(context.Context, platformStorage.MoveFileInput) error {
	return nil
}
func (m *memStore) PresignUpload(context.Context, platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, nil
}
func (m *memStore) PresignDownload(context.Context, platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, nil
}

func TestTenantObjectsRejectsForeignKeys(t *testing.T) {
	key := "1-day/tenants/acme/uploads/catalog/user-1/file.csv"
	store := &memStore{data: map[string][]byte{key: []byte("csv")}}
	objects := application.NewTenantObjects(store)
	ctx := tenant.WithTenantID(context.Background(), "acme")

	got, err := objects.Open(ctx, "catalog", key, 0)
	require.NoError(t, err)
	defer got.Body.Close()
	assert.Equal(t, int64(3), got.SizeBytes)

	_, err = objects.Open(ctx, "catalog", "1-day/tenants/other/uploads/catalog/user-1/file.csv", 0)
	assert.Error(t, err)
	_, err = objects.Open(ctx, "invoices", key, 0)
	assert.Error(t, err)

	sliced, err := objects.Open(ctx, "catalog", key, 1)
	require.NoError(t, err)
	defer sliced.Body.Close()
	body, err := io.ReadAll(sliced.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("sv"), body)
}
