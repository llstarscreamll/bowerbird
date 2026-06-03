package storage

import (
	"context"
	"time"
)

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

type ExistsFileInput struct {
	Path string
}

type MoveFileInput struct {
	SourcePath      string
	DestinationPath string
}

type PresignUploadInput struct {
	Path        string
	ContentType string
	Metadata    map[string]string
	ExpiresIn   time.Duration
}

type PresignDownloadInput struct {
	Path      string
	ExpiresIn time.Duration
}

type FileReference struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

type PresignUploadResult struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Reference  FileReference     `json:"reference"`
	UploadPath string            `json:"upload_path"`
}

type PresignDownloadResult struct {
	URL       string        `json:"url"`
	Method    string        `json:"method"`
	ExpiresAt time.Time     `json:"expires_at"`
	Reference FileReference `json:"reference"`
}

type FileStore interface {
	WriteFileIfAbsent(ctx context.Context, input WriteFileIfAbsentInput) (*WriteFileIfAbsentResult, error)
	ReadFile(ctx context.Context, input ReadFileInput) ([]byte, error)
	Exists(ctx context.Context, input ExistsFileInput) (bool, error)
	MoveFile(ctx context.Context, input MoveFileInput) error
	PresignUpload(ctx context.Context, input PresignUploadInput) (*PresignUploadResult, error)
	PresignDownload(ctx context.Context, input PresignDownloadInput) (*PresignDownloadResult, error)
}
