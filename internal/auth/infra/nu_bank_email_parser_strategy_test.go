package infra

import (
	"encoding/base64"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var nuTransferToNuEmailMessage domain.MailMessage
var nuAccountStatementEmailMessage domain.MailMessage

func TestNuToNuTransfer(t *testing.T) {
	initSampleData()

	input := nuTransferToNuEmailMessage

	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input, []string{})

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

func TestShouldReturnTransactionsFromNuAccountStatement(t *testing.T) {
	initSampleData()

	input := nuAccountStatementEmailMessage

	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input, []string{"bad-password", "1057581292"})

	assert.Equal(t, 112, len(result), "transactions count")

	incomes := slices.DeleteFunc(result, func(t domain.Transaction) bool {
		return t.Type == "expense"
	})
	assert.Equal(t, 3, len(incomes), "incomes count")
	assert.Equal(t, 109, len(result)-len(incomes), "expenses count")
}

func TestShouldReturnNoTransactionsFromNuAccountStatementIfPasswordsAreEmpty(t *testing.T) {
	initSampleData()

	input := nuAccountStatementEmailMessage

	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input, []string{})

	assert.Equal(t, 0, len(result), "transactions count")
}

func TestShouldReturnNoTransactionsFromNuAccountStatementIfPasswordsAreWrong(t *testing.T) {
	initSampleData()

	input := nuAccountStatementEmailMessage

	strategy := &NuBankEmailParserStrategy{}
	result := strategy.Parse(input, []string{"bad-password", "another-bad-password"})

	assert.Equal(t, 0, len(result), "transactions count")
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

	html, err = os.ReadFile("../testdata/nu_savings_account_statement_mail.html")
	if err != nil {
		log.Fatal(err)
	}

	pdf, err := os.ReadFile("../testdata/nu_savings_account_statement.pdf")
	if err != nil {
		log.Fatal(err)
	}

	nuAccountStatementEmailMessage = domain.MailMessage{
		ID:         "email-id-02",
		ExternalID: "email-external-id-02",
		UserID:     "user-id-02",
		From:       "nu@nu.com.co",
		To:         "jhon.doe@gmail.com",
		Subject:    "El extracto de tu cuenta Nu ya está aquí",
		Body:       string(html),
		ReceivedAt: time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC),
		Attachments: []domain.MailAttachment{
			{
				Name:        "CuentaNu_YAC292_2025-04.pdf",
				Content:     base64.StdEncoding.EncodeToString(pdf),
				ContentType: "application/pdf",
			},
		},
	}
}
