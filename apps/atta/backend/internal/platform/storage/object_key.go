package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// InboxAttachmentObjectKey builds the object key for a synced inbox attachment.
// storageFileID must be an internal opaque ID (ULID), not a provider attachment ID.
func InboxAttachmentObjectKey(tenantID, connectionID, messageID, storageFileID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf(
		"tenant/%s/inbox/%s/messages/%s/attachments/%s%s",
		tenantID,
		connectionID,
		messageID,
		storageFileID,
		ext,
	)
}
