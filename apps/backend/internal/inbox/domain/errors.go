package domain

import "errors"

var (
	ErrEmailMessageIDRequired           = errors.New("email message ID is required")
	ErrEmailMessageAccountIDRequired    = errors.New("email message account ID is required")
	ErrEmailMessageProviderIDRequired   = errors.New("email message provider ID is required")
	ErrEmailAttachmentIDRequired        = errors.New("email attachment ID is required")
	ErrEmailAttachmentMessageIDRequired = errors.New("email attachment message ID is required")
	ErrEmailAttachmentFilenameRequired  = errors.New("email attachment filename is required")
	ErrEmailAttachmentSHARequired       = errors.New("email attachment SHA256 is required")
	ErrEmailAttachmentS3KeyRequired     = errors.New("email attachment S3 key is required")
)
