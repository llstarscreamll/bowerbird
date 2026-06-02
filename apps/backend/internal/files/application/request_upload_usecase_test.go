package application

import (
	"context"
	"testing"
	"time"

	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type fakeFileStore struct {
	presignInput platformstorage.PresignUploadInput
}

func (f *fakeFileStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	return nil, nil
}

func (f *fakeFileStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}

func (f *fakeFileStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	return false, nil
}

func (f *fakeFileStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	return nil
}

func (f *fakeFileStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	f.presignInput = input
	return &platformstorage.PresignUploadResult{URL: "https://example.test/upload", Method: "PUT", UploadPath: input.Path}, nil
}

func TestRequestUploadURLUseCaseBuildsTenantUserScopedPrefix(t *testing.T) {
	store := &fakeFileStore{}
	uc := NewRequestUploadURLUseCase(store)

	_, err := uc.Execute(context.Background(), RequestUploadURLCommand{
		TenantID:    "tenant-a",
		UserID:      "user-b",
		Filename:    "invoice.xml",
		ContentType: "application/xml",
		Module:      "invoicing",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if store.presignInput.Path == "" {
		t.Fatal("expected generated upload path")
	}
	if got := store.presignInput.Path; len(got) < len("1-day/tenant-a/uploads/user-b/") || got[:len("1-day/tenant-a/uploads/user-b/")] != "1-day/tenant-a/uploads/user-b/" {
		t.Fatalf("expected tenant/user scoped path prefix, got %s", store.presignInput.Path)
	}
	if store.presignInput.ContentType != "application/xml" {
		t.Fatalf("unexpected content type: %s", store.presignInput.ContentType)
	}
	if store.presignInput.ExpiresIn != 15*time.Minute {
		t.Fatalf("unexpected expiration: %s", store.presignInput.ExpiresIn)
	}
	if store.presignInput.Metadata["module"] != "invoicing" {
		t.Fatalf("unexpected module metadata: %s", store.presignInput.Metadata["module"])
	}
}
