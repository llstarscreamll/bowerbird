package domain

type EmailParserStrategy interface {
	Parse(message MailMessage, passwords []string) []Transaction
}
