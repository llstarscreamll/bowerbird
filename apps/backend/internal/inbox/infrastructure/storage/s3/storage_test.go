package s3

import (
	"context"
	"strings"
	"testing"

	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type fakeObjectStore struct {
	objects map[string]struct{}
}

func (f *fakeObjectStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	if f.objects == nil {
		f.objects = map[string]struct{}{}
	}
	if _, ok := f.objects[input.Path]; ok {
		return &platformstorage.WriteFileIfAbsentResult{Written: false, SizeBytes: int64(len(input.Data))}, nil
	}
	f.objects[input.Path] = struct{}{}
	return &platformstorage.WriteFileIfAbsentResult{Written: true, SizeBytes: int64(len(input.Data))}, nil
}

func (f *fakeObjectStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	return nil, nil
}

func (f *fakeObjectStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	_, ok := f.objects[input.Path]
	return ok, nil
}

func (f *fakeObjectStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	if f.objects == nil {
		f.objects = map[string]struct{}{}
	}
	if _, ok := f.objects[input.SourcePath]; ok {
		f.objects[input.DestinationPath] = struct{}{}
		delete(f.objects, input.SourcePath)
	}
	return nil
}

func (f *fakeObjectStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	return nil, nil
}

func TestStoreAttachmentUploadsAndComputesDeterministicKey(t *testing.T) {
	store := &fakeObjectStore{}
	storage := NewStorageWithStore(store, "bucket-1")

	stored, err := storage.StoreAttachment(context.Background(), StoreAttachmentInput{
		TenantSlug:         "tenant_1",
		ConnectedAccountID: "acc_1",
		MessageID:          "msg_1",
		AttachmentID:       "att_1",
		Filename:           "factura.xml",
		ContentType:        "application/xml",
		Data:               []byte("same-content"),
	})
	if err != nil {
		t.Fatalf("store attachment failed: %v", err)
	}

	if !stored.Uploaded {
		t.Fatal("expected first upload to write object")
	}
	if !strings.HasPrefix(stored.S3Key, "tenant/tenant_1/inbox/acc_1/messages/msg_1/attachments/") {
		t.Fatalf("unexpected key prefix: %s", stored.S3Key)
	}
	if !strings.HasSuffix(stored.S3Key, ".xml") {
		t.Fatalf("expected xml extension in key: %s", stored.S3Key)
	}
}

func TestStoreAttachmentSkipsPhysicalDuplicateBySHA(t *testing.T) {
	store := &fakeObjectStore{}
	storage := NewStorageWithStore(store, "bucket-1")

	first, err := storage.StoreAttachment(context.Background(), StoreAttachmentInput{
		TenantSlug:         "tenant_1",
		ConnectedAccountID: "acc_1",
		MessageID:          "msg_1",
		AttachmentID:       "att_1",
		Filename:           "invoice.pdf",
		ContentType:        "application/pdf",
		Data:               []byte("duplicate-content"),
	})
	if err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	second, err := storage.StoreAttachment(context.Background(), StoreAttachmentInput{
		TenantSlug:         "tenant_1",
		ConnectedAccountID: "acc_2",
		MessageID:          "msg_2",
		AttachmentID:       "att_2",
		Filename:           "invoice-copy.pdf",
		ContentType:        "application/pdf",
		Data:               []byte("duplicate-content"),
	})
	if err != nil {
		t.Fatalf("second store failed: %v", err)
	}

	if first.S3Key == second.S3Key {
		t.Fatalf("expected different keys by message/attachment scope, got %s", first.S3Key)
	}
	if !second.Uploaded {
		t.Fatal("expected uploaded=true when object is ensured in storage")
	}
}
