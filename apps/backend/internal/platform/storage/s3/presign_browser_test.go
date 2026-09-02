package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bowerbird/internal/platform/awsconfig"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	platformS3 "github.com/bowerbird/internal/platform/storage/s3"
)

func TestPresignUploadViaMediaProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	endpoint := "http://localhost:9000"
	presignEndpoint := "https://media.bowerbird.dev"
	cfg, err := awsconfig.Load(ctx, "us-east-1", endpoint, "bowerbird", "bowerbirdsecret")
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	s3Client := awsconfig.NewS3Client(cfg, endpoint)
	presignClient := awsconfig.NewS3PresignClient(cfg, presignEndpoint)
	store := platformS3.NewObjectStoreWithClients(s3Client, presignClient, "bowerbird-local-bucket")

	metadata := map[string]string{
		"tenant-id":           "t1",
		"user-id":             "u1",
		"module":              "invoices",
		"original-filename":   "test.zip",
		"content-type":        "application/zip",
		"cache-control":       "max-age=31536000, public, immutable",
		"content-disposition": `attachment; filename="test.zip"`,
	}

	result, err := store.PresignUpload(ctx, platformStorage.PresignUploadInput{
		Path:        "1-day/tenants/t1/uploads/invoices/u1/test-presign.zip",
		ContentType: "application/zip",
		Metadata:    metadata,
	})
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}
	if !strings.Contains(result.URL, "media.bowerbird.dev") {
		t.Fatalf("expected media host in URL, got %s", result.URL)
	}

	t.Run("without metadata headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, result.URL, bytes.NewReader([]byte("abc")))
		req.Header.Set("Content-Type", "application/zip")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("put request: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("status=%d body=%s", resp.StatusCode, body)
	})

	t.Run("with presign response headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, result.URL, bytes.NewReader([]byte("abc")))
		for k, v := range result.Headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("put request: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("status=%d body=%s", resp.StatusCode, body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if len(result.Headers) == 0 || result.Headers["x-amz-meta-tenant-id"] == "" {
			t.Fatalf("expected signed metadata headers in presign response, got %#v", result.Headers)
		}
	})
}
