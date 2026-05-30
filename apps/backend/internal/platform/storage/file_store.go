package storage

import "context"

type WriteFileIfAbsentInput struct {
	Path        string
	Data        []byte
	ContentType string
	Metadata    map[string]string
}

type WriteFileIfAbsentResult struct {
	Written   bool
	SizeBytes int64
}

type ReadFileInput struct {
	Path string
}

type FileStore interface {
	WriteFileIfAbsent(ctx context.Context, input WriteFileIfAbsentInput) (*WriteFileIfAbsentResult, error)
	ReadFile(ctx context.Context, input ReadFileInput) ([]byte, error)
}
