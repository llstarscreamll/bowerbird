package api

import "context"

type AttachmentRef struct {
	S3Key    string
	Filename string
	MimeType string
}

type ExtractionCandidate struct {
	MessageID   string
	Subject     string
	Snippet     string
	Attachments []AttachmentRef
}

type ExtractionCandidatePage struct {
	Items      []ExtractionCandidate
	NextCursor string
}

type InvoiceBackfillSource interface {
	ListExtractionCandidates(ctx context.Context, cursor string, limit int) (ExtractionCandidatePage, error)
}
