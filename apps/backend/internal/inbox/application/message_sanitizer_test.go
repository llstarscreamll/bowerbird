package application

import (
	"testing"

	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

func TestSanitizeAndValidateMailMessage_SanitizesFields(t *testing.T) {
	message := &domain.MailMessage{
		Subject:       "<b>Factura</b> &amp; alerta",
		Snippet:       "<div>Hola<script>alert(1)</script></div>",
		PlainTextBody: "linea\n\tuno\x00",
		Sender:        "Test User <USER@Example.COM>",
	}

	err := sanitizeAndValidateMailMessage(message)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if message.Subject != "Factura & alerta" {
		t.Fatalf("unexpected sanitized subject: %q", message.Subject)
	}
	if message.Snippet != "Hola alert(1)" {
		t.Fatalf("unexpected sanitized snippet: %q", message.Snippet)
	}
	if message.Sender != "user@example.com" {
		t.Fatalf("unexpected normalized sender: %q", message.Sender)
	}
}

func TestSanitizeAndValidateMailMessage_RejectsMaliciousSender(t *testing.T) {
	message := &domain.MailMessage{
		Sender: `<img src=x onerror=alert(1)>`,
	}

	err := sanitizeAndValidateMailMessage(message)
	if err == nil {
		t.Fatalf("expected sender validation error")
	}
}
