package commands

import (
	"errors"
	"fmt"

	"github.com/bowerbird/internal/connections/domain"
)

var ErrCipherNotConfigured = errors.New("credentials cipher is not configured")

type CredentialsCipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type CredentialsService struct {
	cipher CredentialsCipher
}

func NewCredentialsService(cipher CredentialsCipher) *CredentialsService {
	return &CredentialsService{cipher: cipher}
}

func (s *CredentialsService) EncryptForStorage(plaintext []byte) ([]byte, error) {
	if s.cipher == nil {
		return nil, ErrCipherNotConfigured
	}

	encrypted, err := s.cipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	return encrypted, nil
}

func (s *CredentialsService) DecryptFromStorage(ciphertext []byte) ([]byte, error) {
	if s.cipher == nil {
		return nil, ErrCipherNotConfigured
	}

	plaintext, err := s.cipher.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	return plaintext, nil
}

func (s *CredentialsService) SetEncryptedCredentials(account *domain.Connection, plaintext []byte) error {
	encrypted, err := s.EncryptForStorage(plaintext)
	if err != nil {
		return err
	}

	account.EncryptedCredentials = encrypted
	return nil
}

func (s *CredentialsService) ReadDecryptedCredentials(account *domain.Connection) ([]byte, error) {
	return s.DecryptFromStorage(account.EncryptedCredentials)
}
