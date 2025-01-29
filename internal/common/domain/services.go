package domain

import "time"

type ULIDGenerator interface {
	New() string
	NewFromDate(time.Time) (string, error)
}

type Crypt interface {
	EncryptString(str string) (string, error)
	DecryptString(text string) (string, error)
}
