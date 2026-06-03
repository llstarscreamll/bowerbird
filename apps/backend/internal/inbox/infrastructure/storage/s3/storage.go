package s3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
	platforms3 "github.com/money-path/bowerbird/apps/backend/internal/platform/storage/s3"
)

type Storage struct {
	store  objectStore
	bucket string
}

type objectStore interface {
	platformStorage.FileStore
}

type StoreAttachmentInput struct {
	TenantSlug         string
	ConnectedAccountID string
	MessageID          string
	AttachmentID       string
	Filename           string
	ContentType        string
	Data               []byte
}

type StoredAttachment struct {
	S3Key     string
	SHA256    string
	SizeBytes int64
	Uploaded  bool
}

func NewStorage(client *awsS3.Client, bucket string) *Storage {
	return &Storage{store: platforms3.NewObjectStore(client, bucket), bucket: bucket}
}

func NewStorageWithStore(store objectStore, bucket string) *Storage {
	return &Storage{store: store, bucket: bucket}
}

func (s *Storage) StoreAttachment(ctx context.Context, input StoreAttachmentInput) (*StoredAttachment, error) {
	if s.store == nil {
		return nil, fmt.Errorf("s3 object store is required")
	}
	if s.bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if input.TenantSlug == "" {
		return nil, fmt.Errorf("tenant_slug is required")
	}
	if input.ConnectedAccountID == "" {
		return nil, fmt.Errorf("connected_account_id is required")
	}
	if input.MessageID == "" {
		return nil, fmt.Errorf("message_id is required")
	}
	if input.AttachmentID == "" {
		return nil, fmt.Errorf("attachment_id is required")
	}
	if len(input.Data) == 0 {
		return nil, fmt.Errorf("attachment data is required")
	}

	hash := sha256.Sum256(input.Data)
	hashHex := hex.EncodeToString(hash[:])
	key := buildS3Key(input.TenantSlug, input.ConnectedAccountID, input.MessageID, input.AttachmentID, input.Filename)

	res, err := s.store.WriteFileIfAbsent(ctx, platformStorage.WriteFileIfAbsentInput{
		Path:        key,
		Data:        input.Data,
		ContentType: input.ContentType,
		Metadata: map[string]string{
			"tenant_slug":          input.TenantSlug,
			"connected_account_id": input.ConnectedAccountID,
			"message_id":           input.MessageID,
			"attachment_id":        input.AttachmentID,
			"sha256":               hashHex,
			"orig_name":            safeMetadata(input.Filename),
			"module":               "inbox",
			"stage":                "raw",
		},
	})
	if err != nil {
		return nil, err
	}

	return &StoredAttachment{S3Key: key, SHA256: hashHex, SizeBytes: res.SizeBytes, Uploaded: res.Written}, nil
}

func buildS3Key(tenantSlug, connectedAccountID, messageID, attachmentID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf(
		"tenant/%s/inbox/%s/messages/%s/attachments/%s%s",
		tenantSlug,
		connectedAccountID,
		messageID,
		attachmentID,
		ext,
	)
}

func safeMetadata(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "unknown"
	}
	if len(v) > 256 {
		return v[:256]
	}
	return v
}
