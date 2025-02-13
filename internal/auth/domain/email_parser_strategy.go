package domain

type EmailParserStrategy interface {
	Parse(message MailMessage) []Transaction
}
