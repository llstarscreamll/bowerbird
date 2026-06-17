package id

import (
	"strings"

	"github.com/oklog/ulid/v2"
)

func NewULID() string {
	return ulid.Make().String()
}

func IsValidULID(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	_, err := ulid.ParseStrict(normalized)
	return err == nil
}
