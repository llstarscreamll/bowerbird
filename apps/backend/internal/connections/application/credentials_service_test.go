package application

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/domain"
	platformcrypto "github.com/money-path/bowerbird/apps/backend/internal/platform/crypto"
)

func TestCredentialsServiceEncryptDecryptRoundtrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	cipher, err := platformcrypto.NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}

	svc := NewCredentialsService(cipher)
	plaintext := []byte(`{"access_token":"token","refresh_token":"refresh"}`)

	encrypted, err := svc.EncryptForStorage(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("expected encrypted bytes to differ from plaintext")
	}

	decoded, err := svc.DecryptFromStorage(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("expected %s, got %s", string(plaintext), string(decoded))
	}
}

func TestCredentialsServiceSetAndReadOnConnection(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	cipher, err := platformcrypto.NewAESCipherFromBase64Key(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}

	svc := NewCredentialsService(cipher)
	account := &domain.Connection{ID: "01JWTENANTACCOUNT1234567890"}
	plaintext := []byte(`{"refresh_token":"rt_123"}`)

	if err := svc.SetEncryptedCredentials(account, plaintext); err != nil {
		t.Fatalf("set encrypted credentials failed: %v", err)
	}

	if len(account.EncryptedCredentials) == 0 {
		t.Fatal("expected encrypted credentials to be set")
	}

	decoded, err := svc.ReadDecryptedCredentials(account)
	if err != nil {
		t.Fatalf("read decrypted credentials failed: %v", err)
	}

	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("expected %s, got %s", string(plaintext), string(decoded))
	}
}

func TestCredentialsServiceFailsWhenCipherNotConfigured(t *testing.T) {
	svc := NewCredentialsService(nil)
	if _, err := svc.EncryptForStorage([]byte("x")); err == nil {
		t.Fatal("expected error when cipher is not configured")
	}

	if _, err := svc.DecryptFromStorage([]byte("x")); err == nil {
		t.Fatal("expected error when cipher is not configured")
	}
}
