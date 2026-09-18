package id

import (
	"strings"
	"testing"
)

func TestIsValidULID(t *testing.T) {
	valid := NewULID()

	if !IsValidULID(valid) {
		t.Fatal("expected generated ULID to be valid")
	}

	if !IsValidULID(strings.ToLower(valid)) {
		t.Fatal("expected lowercase ULID to be valid")
	}
}

func TestIsValidULIDRejectsInvalidValue(t *testing.T) {
	if IsValidULID("not-a-ulid") {
		t.Fatal("expected invalid ULID to be rejected")
	}
}
