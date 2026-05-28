package application

import (
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func sanitizeAndValidateMailMessage(message *domain.MailMessage) error {
	if message == nil {
		return fmt.Errorf("provider message is nil")
	}

	message.Subject = sanitizeTextField(message.Subject)
	message.Snippet = sanitizeTextField(message.Snippet)
	message.PlainTextBody = sanitizeTextField(message.PlainTextBody)

	if strings.TrimSpace(message.Sender) == "" {
		return nil
	}

	rawSender := strings.TrimSpace(message.Sender)
	lowerSender := strings.ToLower(rawSender)
	if strings.Contains(lowerSender, "<script") ||
		strings.Contains(lowerSender, "javascript:") ||
		strings.Contains(lowerSender, "onerror=") ||
		strings.Contains(lowerSender, "onload=") {
		return fmt.Errorf("sender contains disallowed markup")
	}

	addr, err := mail.ParseAddress(rawSender)
	if err != nil {
		return fmt.Errorf("invalid sender format: %w", err)
	}

	message.Sender = strings.ToLower(strings.TrimSpace(addr.Address))
	return nil
}

func sanitizeTextField(value string) string {
	cleaned := html.UnescapeString(value)
	cleaned = htmlTagPattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, cleaned)

	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}

	return strings.Join(strings.Fields(cleaned), " ")
}
