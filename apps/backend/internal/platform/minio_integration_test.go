package platform_test

import (
	"context"
	"os"
	"testing"

	"github.com/bowerbird/internal/platform"
	"github.com/bowerbird/internal/platform/id"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

func TestPlatformModuleWritesToMinIO(t *testing.T) {
	if os.Getenv("MINIO_ENDPOINT_URL") == "" {
		t.Skip("MINIO_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	deps, err := platform.NewModule(ctx)
	if err != nil {
		t.Fatalf("new module: %v", err)
	}
	defer deps.ControlDB.Close()
	defer deps.TenantRegistry.CloseAll()

	fileID := id.NewULID()
	key := platformStorage.InboxAttachmentObjectKey(
		"01M1FVPV7SVFDAHHY6919VJ7JM",
		"acc-conn-1",
		"01M1GVRNBM28D1PCWJTJF7T4GF",
		fileID,
		"factura.pdf",
	)

	_, err = deps.FileStore.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
		Path:        key,
		Data:        []byte("integration-payload"),
		ContentType: "application/pdf",
		Metadata: map[string]string{
			"tenant_id":           "01M1FVPV7SVFDAHHY6919VJ7JM",
			"connection_id":       "acc-conn-1",
			"provider_message_id": "gmail-msg",
			"message_id":          "01M1GVRNBM28D1PCWJTJF7T4GF",
			"sha256":              "abc",
			"orig_name":           "Factura N° 123.pdf",
			"module":              "inbox",
			"stage":               "raw",
		},
	})
	if err != nil {
		t.Fatalf("write via platform module: %v", err)
	}
}
