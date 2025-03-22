package infra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

type GoCrypt struct {
	key string
}

func (c GoCrypt) EncryptString(text string) (string, error) {
	plainText := []byte(text)
	block, err := aes.NewCipher([]byte(c.key))
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	cipherText := gcm.Seal(nonce, nonce, plainText, nil)
	encodedCipherText := base64.StdEncoding.EncodeToString(cipherText)

	return string(encodedCipherText), nil
}

func (c GoCrypt) DecryptString(text string) (string, error) {
	decodedCipherText, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(c.key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(decodedCipherText) < nonceSize {
		return "", err
	}

	nonce, cipherText := decodedCipherText[:nonceSize], decodedCipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

func NewGoCrypt(key string) *GoCrypt {
	return &GoCrypt{key}
}
