package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestAESCipherEncryptDecryptRoundtrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	c, err := NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}

	plaintext := []byte(`{"refresh_token":"abc123","expiry":"2026-01-01T00:00:00Z"}`)
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decoded, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("roundtrip mismatch: expected %s, got %s", string(plaintext), string(decoded))
	}
}

func TestAESCipherRejectsInvalidKey(t *testing.T) {
	if _, err := NewAESCipherFromBase64Key("not-base64"); err == nil {
		t.Fatal("expected key decode error")
	}

	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	if _, err := NewAESCipherFromBase64Key(shortKey); err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func TestAESCipherRejectsTamperedCiphertext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	c, err := NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}

	ciphertext, err := c.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	ciphertext[len(ciphertext)-1] ^= 0x01
	if _, err := c.Decrypt(ciphertext); err == nil {
		t.Fatal("expected decrypt error for tampered payload")
	}
}
