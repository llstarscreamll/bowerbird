package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const gcmNonceSize = 12

var (
	ErrEmptyPlaintext  = errors.New("plaintext is required")
	ErrEmptyCiphertext = errors.New("ciphertext is required")
	ErrInvalidKey      = errors.New("invalid encryption key")
)

type AESCipher struct {
	aead cipher.AEAD
}

func NewAESCipherFromBase64Key(encodedKey string) (*AESCipher, error) {
	if encodedKey == "" {
		return nil, ErrInvalidKey
	}

	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	if l := len(key); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("invalid key length %d: %w", l, ErrInvalidKey)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm cipher: %w", err)
	}

	return &AESCipher{aead: aead}, nil
}

func (c *AESCipher) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	sealed := c.aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 0, len(nonce)+len(sealed))
	result = append(result, nonce...)
	result = append(result, sealed...)
	return result, nil
}

func (c *AESCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, ErrEmptyCiphertext
	}

	if len(ciphertext) <= gcmNonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:gcmNonceSize]
	payload := ciphertext[gcmNonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	return plaintext, nil
}
