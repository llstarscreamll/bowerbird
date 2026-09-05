package api

import "context"

type PasswordCandidate struct {
	ID    string
	Value string
}

// DocumentPasswordResolver is the secrets Open Host Service for invoice extraction.
type DocumentPasswordResolver interface {
	ResolveCandidates(ctx context.Context) ([]PasswordCandidate, error)
	MarkUsed(ctx context.Context, secretID string) error
}
