package application

import (
	"fmt"

	"github.com/bowerbird/internal/connections/application/ports"
)

type CredentialsCipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type Credentials struct {
	cipher CredentialsCipher
}

func NewCredentials(cipher CredentialsCipher) *Credentials {
	if cipher == nil {
		panic("credentials cipher is required")
	}
	return &Credentials{cipher: cipher}
}

func (c *Credentials) Encrypt(plaintext []byte) ([]byte, error) {
	encrypted, err := c.cipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}
	return encrypted, nil
}

func (c *Credentials) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := c.cipher.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	return plaintext, nil
}

var _ ports.Credentials = (*Credentials)(nil)
