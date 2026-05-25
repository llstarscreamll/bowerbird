package domain

import "errors"

var (
	ErrNilConnectedAccount              = errors.New("connected account is nil")
	ErrSyncFailureReasonRequired        = errors.New("sync failure reason is required")
	ErrEmailMessageIDRequired           = errors.New("email message id is required")
	ErrEmailMessageAccountIDRequired    = errors.New("email message account id is required")
	ErrEmailMessageProviderIDRequired   = errors.New("email message provider message id is required")
	ErrEmailAttachmentIDRequired        = errors.New("email attachment id is required")
	ErrEmailAttachmentMessageIDRequired = errors.New("email attachment message id is required")
	ErrEmailAttachmentFilenameRequired  = errors.New("email attachment filename is required")
	ErrEmailAttachmentSHARequired       = errors.New("email attachment sha256 is required")
	ErrEmailAttachmentS3KeyRequired     = errors.New("email attachment s3 key is required")
)
