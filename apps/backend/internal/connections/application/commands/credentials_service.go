package commands

import (
	"fmt"

	"github.com/bowerbird/internal/connections/domain"
)

type CredentialsCipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type CredentialsService struct {
	cipher CredentialsCipher
}

func NewCredentialsService(cipher CredentialsCipher) *CredentialsService {
	if cipher == nil {
		panic("credentials cipher is required")
	}
	return &CredentialsService{cipher: cipher}
}

func (s *CredentialsService) EncryptForStorage(plaintext []byte) ([]byte, error) {
	encrypted, err := s.cipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	return encrypted, nil
}

func (s *CredentialsService) DecryptFromStorage(ciphertext []byte) ([]byte, error) {
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

	return account.AssignEncryptedCredentials(encrypted)
}

func (s *CredentialsService) ReadDecryptedCredentials(account *domain.Connection) ([]byte, error) {
	return s.DecryptFromStorage(account.EncryptedCredentials)
}
