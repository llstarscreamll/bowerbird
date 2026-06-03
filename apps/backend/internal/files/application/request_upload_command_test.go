package application

import (
	"context"
	"strings"
	"testing"
	"time"

	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFileStore struct {
	presignInput platformStorage.PresignUploadInput
}

func (f *fakeFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}

func (f *fakeFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}

func (f *fakeFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	return false, nil
}

func (f *fakeFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (f *fakeFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	f.presignInput = input
	return &platformStorage.PresignUploadResult{URL: "https://example.test/upload", Method: "PUT", UploadPath: input.Path}, nil
}

func (f *fakeFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	return nil, nil
}

func TestRequestUploadURLUseCaseBuildsTenantUserModuleScopedPrefix(t *testing.T) {
	store := &fakeFileStore{}
	uc := NewRequestUploadURLCommand(store)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	_, err := uc.Execute(ctx, RequestUploadURLInput{
		UserID:      "user-b",
		Filename:    "invoice.xml",
		ContentType: "application/xml",
		Module:      "invoicing",
	})
	require.NoError(t, err)

	require.NotEmpty(t, store.presignInput.Path)
	assert.Contains(t, store.presignInput.Path, "1-day/tenants/tenant-a/uploads/invoicing/user-b/")
	assert.True(t, strings.HasSuffix(store.presignInput.Path, ".xml"))
	assert.Equal(t, "application/xml", store.presignInput.ContentType)
	assert.Equal(t, 60*time.Minute, store.presignInput.ExpiresIn)
	assert.Equal(t, "invoicing", store.presignInput.Metadata["module"])
	assert.Equal(t, "tenant-a", store.presignInput.Metadata["tenant-id"])
	assert.Equal(t, "user-b", store.presignInput.Metadata["user-id"])
	assert.Equal(t, "invoice.xml", store.presignInput.Metadata["original-filename"])
	assert.Equal(t, "application/xml", store.presignInput.Metadata["content-type"])
	assert.Equal(t, "max-age=31536000, public, immutable", store.presignInput.Metadata["cache-control"])
}
