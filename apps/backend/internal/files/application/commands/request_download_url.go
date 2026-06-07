package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformStorage "github.com/bowerbird/internal/platform/storage"
	"github.com/bowerbird/internal/platform/tenant"
)

const defaultDownloadURLExpiry = 60 * time.Minute

type RequestDownloadURLInput struct {
	Key string
}

type RequestDownloadURLCommand struct {
	fileStore platformStorage.FileStore
}

func NewRequestDownloadURLCommand(fileStore platformStorage.FileStore) *RequestDownloadURLCommand {
	return &RequestDownloadURLCommand{fileStore: fileStore}
}

func (cmd *RequestDownloadURLCommand) Execute(ctx context.Context, input RequestDownloadURLInput) (*platformStorage.PresignDownloadResult, error) {
	if cmd.fileStore == nil {
		return nil, fmt.Errorf("file store is required")
	}

	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant id from context: %w", err)
	}

	if strings.TrimSpace(input.Key) == "" {
		return nil, fmt.Errorf("file key is required")
	}

	if !cmd.isTenantScopedKey(tenantID, input.Key) {
		return nil, fmt.Errorf("file key is outside tenant scope")
	}

	exists, err := cmd.fileStore.Exists(ctx, platformStorage.ExistsFileInput{Path: input.Key})
	if err != nil {
		return nil, fmt.Errorf("check file existence: %w", err)
	}
	if !exists {
		return nil, ErrFileNotFound
	}

	return cmd.fileStore.PresignDownload(ctx, platformStorage.PresignDownloadInput{
		Path:      input.Key,
		ExpiresIn: defaultDownloadURLExpiry,
	})
}

func (cmd *RequestDownloadURLCommand) isTenantScopedKey(tenantID, key string) bool {
	tenantID = strings.TrimSpace(tenantID)
	key = strings.TrimSpace(key)
	if tenantID == "" || key == "" {
		return false
	}

	prefixes := []string{
		"tenant/" + tenantID + "/",
		"1-day/" + tenantID + "/",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}
