package application

import (
	"context"
	"errors"
	"testing"

	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDownloadFileStore struct {
	exists           bool
	existsInput      platformStorage.ExistsFileInput
	presignInput     platformStorage.PresignDownloadInput
	presignResultURL string
}

func (f *fakeDownloadFileStore) WriteFileIfAbsent(ctx context.Context, input platformStorage.WriteFileIfAbsentInput) (*platformStorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}

func (f *fakeDownloadFileStore) ReadFile(ctx context.Context, input platformStorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}

func (f *fakeDownloadFileStore) Exists(ctx context.Context, input platformStorage.ExistsFileInput) (bool, error) {
	f.existsInput = input
	return f.exists, nil
}

func (f *fakeDownloadFileStore) MoveFile(ctx context.Context, input platformStorage.MoveFileInput) error {
	return nil
}

func (f *fakeDownloadFileStore) PresignUpload(ctx context.Context, input platformStorage.PresignUploadInput) (*platformStorage.PresignUploadResult, error) {
	return nil, nil
}

func (f *fakeDownloadFileStore) PresignDownload(ctx context.Context, input platformStorage.PresignDownloadInput) (*platformStorage.PresignDownloadResult, error) {
	f.presignInput = input
	if f.presignResultURL == "" {
		f.presignResultURL = "https://example.test/download"
	}
	return &platformStorage.PresignDownloadResult{URL: f.presignResultURL, Method: "GET"}, nil
}

func TestRequestDownloadURLUseCasePresignsOnlyTenantScopedKey(t *testing.T) {
	store := &fakeDownloadFileStore{exists: true}
	uc := NewRequestDownloadURLCommand(store)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	result, err := uc.Execute(ctx, RequestDownloadURLInput{
		Key: "1-day/tenant-a/uploads/user-x/invoice.pdf",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.URL)
	assert.Equal(t, "1-day/tenant-a/uploads/user-x/invoice.pdf", store.existsInput.Path)
}

func TestRequestDownloadURLUseCaseRejectsOutOfTenantKey(t *testing.T) {
	store := &fakeDownloadFileStore{exists: true}
	uc := NewRequestDownloadURLCommand(store)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	_, err := uc.Execute(ctx, RequestDownloadURLInput{
		Key: "tenant/tenant-b/inbox/acc/messages/m/attachments/a.pdf",
	})

	require.Error(t, err)
}

func TestRequestDownloadURLUseCaseReturnsNotFoundWhenFileMissing(t *testing.T) {
	store := &fakeDownloadFileStore{exists: false}
	uc := NewRequestDownloadURLCommand(store)
	ctx := tenant.WithTenantID(context.Background(), "tenant-a")

	_, err := uc.Execute(ctx, RequestDownloadURLInput{
		Key: "1-day/tenant-a/uploads/user-x/invoice.pdf",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFileNotFound))
}
