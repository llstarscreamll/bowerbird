package infra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	crypt := NewGoCrypt("YD6Cm1Nemh/+ng2seKxAoQ==")

	input := "some-secret-here"
	encrypted, err := crypt.EncryptString(input)

	assert.Nil(t, err)
	assert.True(t, len(encrypted) > 0, "result should not be empty")
	assert.NotEqual(t, input, encrypted)

	decrypted, err := crypt.DecryptString(encrypted)

	assert.Nil(t, err)
	assert.Equal(t, input, decrypted)
}
