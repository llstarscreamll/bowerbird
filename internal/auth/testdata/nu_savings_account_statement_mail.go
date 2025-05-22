package testdata

import (
	"llstarscreamll/bowerbird/internal/auth/domain"
	"os"
)

var nuSavingsAccountStatementMailHtml, _ = os.ReadFile("nu_savings_account_statement_mail.html")
var nuSavingsAccountStatementPdf, _ = os.ReadFile("nu_savings_account_statement.pdf")

var NuSavingsAccountStatementMail = domain.MailMessage{
	ID:         "email-id-01",
	ExternalID: "email-external-id-01",
	UserID:     "user-id-01",
	From:       "nu@nu.com.co",
	To:         "jhon.doe@gmail.com",
	Subject:    "El extracto de tu cuenta Nu ya está aquí",
	Body:       string(nuSavingsAccountStatementMailHtml),
	Attachments: []domain.MailAttachment{
		{
			Name:        "CuentaNu_YAC292_2025-04.pdf",
			Content:     string(nuSavingsAccountStatementPdf),
			ContentType: "application/pdf",
		},
	},
}
