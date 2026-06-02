package domain

import "time"

type MessageAttachment struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  *string
	SizeBytes *int64
	SHA256    string
	S3Key     string
	RawData   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewMessageAttachmentInput struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  *string
	SizeBytes *int64
	SHA256    string
	S3Key     string
	RawData   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMessageAttachment(input NewMessageAttachmentInput) (*MessageAttachment, error) {
	if input.ID == "" {
		return nil, ErrMessageAttachmentIDRequired
	}
	if input.MessageID == "" {
		return nil, ErrMessageAttachmentMessageIDRequired
	}
	if input.Filename == "" {
		return nil, ErrMessageAttachmentFilenameRequired
	}
	if input.SHA256 == "" {
		return nil, ErrMessageAttachmentSHARequired
	}
	if input.S3Key == "" {
		return nil, ErrMessageAttachmentS3KeyRequired
	}

	return &MessageAttachment{
		ID:        input.ID,
		MessageID: input.MessageID,
		Filename:  input.Filename,
		MimeType:  input.MimeType,
		SizeBytes: input.SizeBytes,
		SHA256:    input.SHA256,
		S3Key:     input.S3Key,
		RawData:   input.RawData,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
	}, nil
}
