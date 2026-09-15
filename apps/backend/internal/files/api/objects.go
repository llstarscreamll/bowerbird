package api

import (
	"context"
	"io"
)

// TenantObjects is the files Open Host Service for reading tenant-owned uploads.
type TenantObjects interface {
	Open(ctx context.Context, module, key string, offset int64) (*Object, error)
}

type Object struct {
	Body      io.ReadCloser
	SizeBytes int64
}
