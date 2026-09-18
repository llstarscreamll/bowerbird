package ports

import (
	"context"
)

type PasswordCandidate struct {
	SecretID string
	Value    string
}

type DocumentPasswordResolver interface {
	ResolveCandidates(ctx context.Context) ([]PasswordCandidate, error)
	MarkUsed(ctx context.Context, secretID string) error
}
