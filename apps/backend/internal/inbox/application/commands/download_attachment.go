package commands

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/bowerbird/internal/inbox/domain"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

type DownloadAttachmentResult struct {
	Filename    string
	ContentType string
	Data        []byte
}

type DownloadAttachmentCommand struct {
	messageRepo domain.MessageRepository
	fileStore   platformStorage.FileStore
}

func NewDownloadAttachmentCommand(messageRepo domain.MessageRepository, fileStore platformStorage.FileStore) *DownloadAttachmentCommand {
	return &DownloadAttachmentCommand{messageRepo: messageRepo, fileStore: fileStore}
}

func (c *DownloadAttachmentCommand) Execute(ctx context.Context, messageID, attachmentID string) (*DownloadAttachmentResult, error) {
	if c.fileStore == nil {
		return nil, fmt.Errorf("file store is required")
	}

	if _, err := c.messageRepo.GetInboxMessageByID(ctx, messageID); err != nil {
		return nil, err
	}

	attachment, err := c.messageRepo.GetMessageAttachment(ctx, messageID, attachmentID)
	if err != nil {
		return nil, err
	}

	data, err := c.fileStore.ReadFile(ctx, platformStorage.ReadFileInput{Path: attachment.S3Key})
	if err != nil {
		return nil, fmt.Errorf("read attachment: %w", err)
	}

	contentType := "application/octet-stream"
	if attachment.MimeType != nil && strings.TrimSpace(*attachment.MimeType) != "" {
		contentType = *attachment.MimeType
	}

	filename := attachment.Filename
	if filename == "" {
		filename = path.Base(attachment.S3Key)
	}

	return &DownloadAttachmentResult{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}
