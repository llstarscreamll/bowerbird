package infra

import (
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/auth/testdata"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNuBankEmailParserStrategy(t *testing.T) {
	expectedDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	input := domain.MailMessage{
		From:       "nu@nu.com.co",
		Subject:    "Tu dinero va en camino",
		Body:       testdata.NuBankTransactionMailExample,
		ReceivedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input)

	assert.Equal(t, 2, len(result))

	assert.Equal(t, "email", result[0].Origin)
	assert.Equal(t, "expense", result[0].Type)
	assert.Equal(t, float32(-300000), result[0].Amount)
	assert.Equal(t, "", result[0].UserDescription)
	assert.Equal(t, "Transferencia a Diana E.", result[0].SystemDescription)
	assert.Equal(t, expectedDate, result[0].ProcessedAt)

	assert.Equal(t, "email", result[1].Origin)
	assert.Equal(t, "expense", result[1].Type)
	assert.Equal(t, float32(-1200), result[1].Amount)
	assert.Equal(t, "", result[1].UserDescription)
	assert.Equal(t, "Impuesto 4x1.000", result[1].SystemDescription)
	assert.Equal(t, expectedDate, result[1].ProcessedAt)
}
