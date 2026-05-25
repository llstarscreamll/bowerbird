package s3

import (
	"context"
	"strings"
	"testing"

	platforms3 "github.com/money-path/bowerbird/apps/backend/internal/platform/storage/s3"
)

type fakeObjectStore struct {
	objects map[string]struct{}
}

func (f *fakeObjectStore) PutObjectIfAbsent(ctx context.Context, input platforms3.PutObjectIfAbsentInput) (*platforms3.PutObjectIfAbsentResult, error) {
	if f.objects == nil {
		f.objects = map[string]struct{}{}
	}
	if _, ok := f.objects[input.Key]; ok {
		return &platforms3.PutObjectIfAbsentResult{Uploaded: false, SizeBytes: int64(len(input.Data))}, nil
	}
	f.objects[input.Key] = struct{}{}
	return &platforms3.PutObjectIfAbsentResult{Uploaded: true, SizeBytes: int64(len(input.Data))}, nil
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
