package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"testing"

	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type fakeFileStore struct {
	data map[string][]byte
}

func (r *fakeFileStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	if r.data == nil {
		r.data = map[string][]byte{}
	}
	if _, ok := r.data[input.Path]; !ok {
		r.data[input.Path] = input.Data
	}
	return &platformstorage.WriteFileIfAbsentResult{Written: true, SizeBytes: int64(len(input.Data))}, nil
}

func (r *fakeFileStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	if payload, ok := r.data[input.Path]; ok {
		return payload, nil
	}
	return nil, errors.New("attachment not found")
}

func (r *fakeFileStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	_, ok := r.data[input.Path]
	return ok, nil
}

func (r *fakeFileStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	r.data[input.DestinationPath] = r.data[input.SourcePath]
	delete(r.data, input.SourcePath)
	return nil
}

func (r *fakeFileStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	return nil, nil
}

func (r *fakeFileStore) PresignDownload(ctx context.Context, input platformstorage.PresignDownloadInput) (*platformstorage.PresignDownloadResult, error) {
	return nil, nil
}

func TestClassifyFromInboxEventGroupsDirectXMLAndPDF(t *testing.T) {
	store := &fakeFileStore{data: map[string][]byte{
		"k1": []byte("<Invoice><ID>INV-1</ID></Invoice>"),
		"k2": []byte("%PDF-1.4 file"),
		"k3": []byte("plain text"),
	}}

	uc := NewClassifyDocumentsUseCase(store)
	res, err := uc.ClassifyFromInboxEvent(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt-1",
		TenantSlug:        "tenant-1",
		AccountID:         "acc-1",
		Provider:          "gmail",
		ProviderMessageID: "msg-1",
		MessageInternalID: "m-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "k1", Filename: "INV-1.xml"},
			{S3Key: "k2", Filename: "INV-1.pdf"},
			{S3Key: "k3", Filename: "notes.txt"},
		},
	})
	if err != nil {
		t.Fatalf("classify failed: %v", err)
	}

	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(res.Groups))
	}
	if res.Groups[0].XML == nil || res.Groups[0].PDF == nil {
		t.Fatalf("expected xml+pdf pair, got %#v", res.Groups[0])
	}
	if len(res.Unclassified) != 1 {
		t.Fatalf("expected 1 unclassified file, got %d", len(res.Unclassified))
	}
}

func TestClassifyFromInboxEventDecompressesZIPAndGroupsPair(t *testing.T) {
	zipPayload := buildZIP(t, map[string][]byte{
		"docs/FE-99.xml":  []byte("<Invoice><ID>FE-99</ID></Invoice>"),
		"docs/FE-99.pdf":  []byte("%PDF-1.4 file"),
		"docs/readme.txt": []byte("hello"),
	})

	store := &fakeFileStore{data: map[string][]byte{
		"zip-key": zipPayload,
	}}

	uc := NewClassifyDocumentsUseCase(store)
	res, err := uc.ClassifyFromInboxEvent(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt-1",
		TenantSlug:        "tenant-1",
		AccountID:         "acc-1",
		Provider:          "gmail",
		ProviderMessageID: "msg-1",
		MessageInternalID: "m-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "zip-key", Filename: "attachments.zip"},
		},
	})
	if err != nil {
		t.Fatalf("classify failed: %v", err)
	}

	if res.CompressedSources != 1 {
		t.Fatalf("expected 1 compressed source, got %d", res.CompressedSources)
	}
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(res.Groups))
	}
	group := res.Groups[0]
	if group.XML == nil || group.PDF == nil {
		t.Fatalf("expected xml+pdf in group, got %#v", group)
	}
	if group.XML.Source != "zip" || group.PDF.Source != "zip" {
		t.Fatalf("expected zip source documents")
	}
	if group.XML.ParentArchive != "attachments.zip" {
		t.Fatalf("expected parent archive traceability, got %q", group.XML.ParentArchive)
	}
}

func TestClassifyFromInboxEventGroupsAcrossDirectAndZipSources(t *testing.T) {
	zipPayload := buildZIP(t, map[string][]byte{
		"INV100.pdf": []byte("%PDF-1.4 file"),
	})

	store := &fakeFileStore{data: map[string][]byte{
		"xml-key": []byte("<Invoice><ID>INV100</ID></Invoice>"),
		"zip-key": zipPayload,
	}}

	uc := NewClassifyDocumentsUseCase(store)
	res, err := uc.ClassifyFromInboxEvent(context.Background(), contractevents.InboxMessageReceived{
		EventID:           "evt-1",
		TenantSlug:        "tenant-1",
		AccountID:         "acc-1",
		Provider:          "gmail",
		ProviderMessageID: "msg-1",
		MessageInternalID: "m-1",
		AttachmentRefs: []contractevents.AttachmentRef{
			{S3Key: "xml-key", Filename: "INV100.xml"},
			{S3Key: "zip-key", Filename: "bundle.zip"},
		},
	})
	if err != nil {
		t.Fatalf("classify failed: %v", err)
	}

	if len(res.Groups) != 1 {
		t.Fatalf("expected single merged group, got %d", len(res.Groups))
	}
	if res.Groups[0].XML == nil || res.Groups[0].PDF == nil {
		t.Fatalf("expected xml+pdf pair from mixed sources")
	}
}

func buildZIP(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip file %s: %v", name, err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("write zip content %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buf.Bytes()
}
