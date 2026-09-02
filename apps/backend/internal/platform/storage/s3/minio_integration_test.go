package s3_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/id"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	platformS3 "github.com/bowerbird/internal/platform/storage/s3"
)

func TestMinIOPutObjectWithSyncMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	endpoint := "http://localhost:9000"
	ctx := context.Background()
	cfg, err := awsconfig.Load(ctx, "us-east-1", endpoint, "bowerbird", "bowerbirdsecret")
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	store := platformS3.NewObjectStore(awsconfig.NewS3Client(cfg, endpoint), "bowerbird-local-bucket")

	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey("01M1FYT1CDMP38H9KAX1JY6STH", "acc-1", "msg-1", fileID, "invoice.pdf")

	_, err = store.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
		Path:        key,
		Data:        []byte("payload"),
		ContentType: "application/pdf",
		Metadata: map[string]string{
			"tenant_id":           "01M1FYT1CDMP38H9KAX1JY6STH",
			"connection_id":       "acc-1",
			"provider_message_id": "gmail-msg-1",
			"message_id":          "msg-1",
			"sha256":              "abc123",
			"orig_name":           "invoice.pdf",
			"module":              "inbox",
			"stage":               "raw",
		},
	})
	if err != nil {
		t.Fatalf("put object with sync metadata: %v", err)
	}
}

func TestMinIOCredentialMismatchErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	endpoint := "http://localhost:9000"
	ctx := context.Background()

	cases := []struct {
		name       string
		accessKey  string
		secretKey  string
		wantSubstr string
	}{
		{name: "wrong secret", accessKey: "bowerbird", secretKey: "wrongsecret", wantSubstr: "Forbidden"},
		{name: "wrong access key", accessKey: "wrong", secretKey: "bowerbirdsecret", wantSubstr: "Forbidden"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := awsconfig.Load(ctx, "us-east-1", endpoint, tc.accessKey, tc.secretKey)
			if err != nil {
				t.Fatalf("load aws config: %v", err)
			}
			store := platformS3.NewObjectStore(awsconfig.NewS3Client(cfg, endpoint), "bowerbird-local-bucket")
			_, err = store.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
				Path:        "debug/credprobe.bin",
				Data:        []byte("payload"),
				ContentType: "application/octet-stream",
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("expected %q in error, got: %v", tc.wantSubstr, err)
			}
		})
	}
}

func TestMinIOLargeAttachmentWithUnicodeMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	endpoint := "http://localhost:9000"
	ctx := context.Background()
	cfg, err := awsconfig.Load(ctx, "us-east-1", endpoint, "bowerbird", "bowerbirdsecret")
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	store := platformS3.NewObjectStore(awsconfig.NewS3Client(cfg, endpoint), "bowerbird-local-bucket")

	data := bytes.Repeat([]byte("x"), 6*1024*1024)
	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey("01M1FVPV7SVFDAHHY6919VJ7JM", "acc-conn-1", "01M1GVRNBM28D1PCWJTJF7T4GF", fileID, "Factura N° 123.pdf")

	_, err = store.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
		Path:        key,
		Data:        data,
		ContentType: "application/pdf",
		Metadata: map[string]string{
			"tenant_id":           "01M1FVPV7SVFDAHHY6919VJ7JM",
			"connection_id":       "acc-conn-1",
			"provider_message_id": "gmail-msg",
			"message_id":          "01M1GVRNBM28D1PCWJTJF7T4GF",
			"sha256":              strings.Repeat("a", 64),
			"orig_name":           "Factura N° 123.pdf",
			"module":              "inbox",
			"stage":               "raw",
		},
	})
	if err != nil {
		t.Fatalf("put large object with unicode metadata: %v", err)
	}
}

func TestMinIOMacOSScreenshotFilenameMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	endpoint := "http://localhost:9000"
	ctx := context.Background()
	cfg, err := awsconfig.Load(ctx, "us-east-1", endpoint, "bowerbird", "bowerbirdsecret")
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	store := platformS3.NewObjectStore(awsconfig.NewS3Client(cfg, endpoint), "bowerbird-local-bucket")

	// macOS screenshots use U+202F (narrow no-break space) before AM/PM.
	filename := "Screenshot 2026-07-08 at 8.12.01\u202fAM.png"
	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey("01M1GWVJCYJFV665R6DKAJCM90", "acc-conn-1", "01M1GXCA0TE3NTPNT8N96WTCFP", fileID, filename)

	_, err = store.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
		Path:        key,
		Data:        bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 512),
		ContentType: "image/png",
		Metadata: map[string]string{
			"tenant_id":           "01M1GWVJCYJFV665R6DKAJCM90",
			"connection_id":       "acc-conn-1",
			"provider_message_id": "19f41dd0a90124a1",
			"message_id":          "01M1GXCA0TE3NTPNT8N96WTCFP",
			"sha256":              strings.Repeat("a", 64),
			"orig_name":           filename,
			"module":              "inbox",
			"stage":               "raw",
		},
	})
	if err != nil {
		t.Fatalf("put object with macOS screenshot filename metadata: %v", err)
	}
}
