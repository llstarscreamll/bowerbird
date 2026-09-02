package storage_test

import (
	"strings"
	"testing"

	"github.com/bowerbird/internal/platform/id"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func TestInboxAttachmentObjectKeyUsesStorageFileID(t *testing.T) {
	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey("tenant-a", "conn-1", "msg-1", fileID, "invoice.xml")
	if !strings.Contains(key, fileID) {
		t.Fatalf("expected storage file id in key, got %q", key)
	}
	if !strings.HasSuffix(key, ".xml") {
		t.Fatalf("expected .xml extension, got %q", key)
	}
	if strings.Contains(key, "gmail-provider-id") {
		t.Fatal("provider attachment id must not appear in object key")
	}
}

func TestInboxAttachmentObjectKeyShortULIDSegment(t *testing.T) {
	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey("t", "a", "m", fileID, "file.pdf")
	segment := strings.TrimSuffix(strings.Split(key, "/attachments/")[1], ".pdf")
	if len(segment) > 255 {
		t.Fatalf("segment too long for MinIO: %d", len(segment))
	}
}
