package commands

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bowerbird/internal/platform/id"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
)

const (
	defaultUploadScope     = "1-day"
	defaultUploadURLExpiry = 60 * time.Minute
)

type RequestUploadURLInput struct {
	UserID      string
	Filename    string
	ContentType string
	Module      string
}

type RequestUploadURLCommand struct {
	fileStore platformStorage.FileStore
}

func NewRequestUploadURLCommand(fileStore platformStorage.FileStore) *RequestUploadURLCommand {
	return &RequestUploadURLCommand{fileStore: fileStore}
}

func (cmd *RequestUploadURLCommand) Execute(ctx context.Context, input RequestUploadURLInput) (*platformStorage.PresignUploadResult, error) {
	if cmd.fileStore == nil {
		return nil, fmt.Errorf("file store is required")
	}

	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant id from context: %w", err)
	}

	if strings.TrimSpace(input.UserID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	if strings.TrimSpace(input.Filename) == "" {
		return nil, fmt.Errorf("filename is required")
	}

	key := cmd.buildPath(tenantID, input.UserID, input.Module, input.Filename)
	return cmd.fileStore.PresignUpload(ctx, platformStorage.PresignUploadInput{
		Path:        key,
		ContentType: input.ContentType,
		ExpiresIn:   defaultUploadURLExpiry,
		Metadata: map[string]string{
			"tenant-id":           tenantID,
			"user-id":             input.UserID,
			"module":              cmd.sanitizeMetadata(input.Module),
			"original-filename":   cmd.sanitizeMetadata(input.Filename),
			"content-type":        input.ContentType,
			"cache-control":       "max-age=31536000, public, immutable",
			"content-disposition": fmt.Sprintf("attachment; filename=\"%s\"", cmd.sanitizeMetadata(input.Filename)),
		},
	})
}

func (cmd *RequestUploadURLCommand) buildPath(tenantID, userID, module, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}

	name := id.NewULID() + ext
	return path.Join(defaultUploadScope, "tenants", tenantID, "uploads", module, userID, name)
}

func (cmd *RequestUploadURLCommand) sanitizeMetadata(value string) string {
	v := strings.TrimSpace(value)
	v = strings.ReplaceAll(v, " ", "_")
	v = url.PathEscape(v)

	if v == "" {
		return "unknown"
	}

	if len(v) > 256 {
		return v[:256]
	}

	return v
}
