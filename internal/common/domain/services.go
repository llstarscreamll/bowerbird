package domain

import (
	"context"
	"time"
)

type ULIDGenerator interface {
	New() string
	NewFromDate(time.Time) (string, error)
}

type Crypt interface {
	EncryptString(str string) (string, error)
	DecryptString(text string) (string, error)
}

type ParameterStore interface {
	GetParameter(ctx context.Context, name string, secure bool) (string, error)
}
