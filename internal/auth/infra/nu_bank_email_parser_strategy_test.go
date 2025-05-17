package infra

import (
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var nuTransferToNuEmailMessage domain.MailMessage

func TestNuToNuTransfer(t *testing.T) {
	initSampleData()

	input := nuTransferToNuEmailMessage

	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input)

	expectedDate := input.ReceivedAt
	assert.Equal(t, 2, len(result), "should return 2 transactions")

	assert.Equal(t, "nu-bank-email", result[0].Origin)
	assert.Equal(t, "expense", result[0].Type)
	assert.Equal(t, float32(-300000), result[0].Amount)
	assert.Equal(t, "", result[0].UserDescription)
	assert.Equal(t, "Envío a Diana E.", result[0].SystemDescription)
	assert.Equal(t, expectedDate, result[0].ProcessedAt)

	assert.Equal(t, "nu-bank-email", result[1].Origin)
	assert.Equal(t, "expense", result[1].Type)
	assert.Equal(t, float32(-1200), result[1].Amount)
	assert.Equal(t, "", result[1].UserDescription)
	assert.Equal(t, "4x1.000", result[1].SystemDescription)
	assert.Equal(t, expectedDate, result[1].ProcessedAt)
}

func initSampleData() {
	html, err := os.ReadFile("../testdata/nu_transfer_to_nu_mail.html")
	if err != nil {
		log.Fatal(err)
	}

	nuTransferToNuEmailMessage = domain.MailMessage{
		ID:          "email-id-01",
		ExternalID:  "email-external-id-01",
		UserID:      "user-id-01",
		From:        "nu@nu.com.co",
		Subject:     "El dinero que enviaste ya está del otro lado",
		To:          "jhon.doe@gmail.com",
		Body:        string(html),
		Attachments: []domain.MailAttachment{},
		ReceivedAt:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
