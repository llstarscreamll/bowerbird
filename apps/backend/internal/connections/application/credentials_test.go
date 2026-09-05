package application

import (
	"bytes"
	"encoding/base64"
	"testing"

	platformcrypto "github.com/bowerbird/internal/platform/crypto"
)

func TestCredentialsEncryptDecryptRoundtrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	cipher, err := platformcrypto.NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}

	creds := NewCredentials(cipher)
	plaintext := []byte(`{"access_token":"token","refresh_token":"refresh"}`)

	encrypted, err := creds.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("expected encrypted bytes to differ from plaintext")
	}

	decoded, err := creds.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("expected %s, got %s", string(plaintext), string(decoded))
	}
}

func TestCredentialsPanicsWhenCipherMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when cipher is missing")
		}
	}()
	NewCredentials(nil)
}
