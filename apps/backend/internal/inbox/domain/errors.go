package domain

import "errors"

var (
	ErrSyncCursorConnectionIDRequired     = errors.New("sync cursor connection ID is required")
	ErrInboxMessageIDRequired             = errors.New("inbox message ID is required")
	ErrInboxMessageConnectionIDRequired   = errors.New("inbox message connection ID is required")
	ErrInboxMessageProviderIDRequired     = errors.New("inbox message provider ID is required")
	ErrMessageAttachmentIDRequired        = errors.New("message attachment ID is required")
	ErrMessageAttachmentMessageIDRequired = errors.New("message attachment message ID is required")
	ErrMessageAttachmentFilenameRequired  = errors.New("message attachment filename is required")
	ErrMessageAttachmentSHARequired       = errors.New("message attachment SHA256 is required")
	ErrMessageAttachmentS3KeyRequired     = errors.New("message attachment S3 key is required")
)
