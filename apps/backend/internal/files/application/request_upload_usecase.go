package application

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

const (
	defaultUploadScope     = "1-day"
	defaultUploadURLExpiry = 15 * time.Minute
)

type RequestUploadURLCommand struct {
	TenantID    string
	UserID      string
	Filename    string
	ContentType string
	Module      string
}

type RequestUploadURLUseCase struct {
	store platformstorage.FileStore
}

func NewRequestUploadURLUseCase(store platformstorage.FileStore) *RequestUploadURLUseCase {
	return &RequestUploadURLUseCase{store: store}
}

func (u *RequestUploadURLUseCase) Execute(ctx context.Context, cmd RequestUploadURLCommand) (*platformstorage.PresignUploadResult, error) {
	if u.store == nil {
		return nil, fmt.Errorf("file store is required")
	}
	if strings.TrimSpace(cmd.TenantID) == "" {
		return nil, fmt.Errorf("tenant id is required")
	}
	if strings.TrimSpace(cmd.UserID) == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(cmd.Filename) == "" {
		return nil, fmt.Errorf("filename is required")
	}

	key := buildUploadPath(cmd.TenantID, cmd.UserID, cmd.Filename)
	return u.store.PresignUpload(ctx, platformstorage.PresignUploadInput{
		Path:        key,
		ContentType: cmd.ContentType,
		ExpiresIn:   defaultUploadURLExpiry,
		Metadata: map[string]string{
			"tenant_id": cmd.TenantID,
			"user_id":   cmd.UserID,
			"module":    sanitizeMetadata(cmd.Module),
			"stage":     "upload",
		},
	})
}

func buildUploadPath(tenantID, userID, filename string) string {
	safeName := sanitizeFilename(filename)
	base := strings.TrimSuffix(safeName, filepath.Ext(safeName))
	ext := strings.ToLower(filepath.Ext(safeName))
	if ext == "" {
		ext = ".bin"
	}

	uniqueName := base + "-" + id.NewULID() + ext
	return path.Join(defaultUploadScope, tenantID, "uploads", userID, uniqueName)
}

func sanitizeFilename(filename string) string {
	base := strings.TrimSpace(filepath.Base(filename))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "file.bin"
	}

	var b strings.Builder
	b.Grow(len(base))
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	safe := strings.Trim(strings.TrimSpace(b.String()), ".-")
	if safe == "" {
		return "file.bin"
	}

	return safe
}

func sanitizeMetadata(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "unknown"
	}
	if len(v) > 256 {
		return v[:256]
	}
	return v
}
