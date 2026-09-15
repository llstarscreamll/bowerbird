package domain

import "testing"

func TestMailMessageNeedsFullContent(t *testing.T) {
	t.Parallel()

	if (&MailMessage{Subject: "Reunión semanal"}).NeedsFullContent() {
		t.Fatal("expected newsletter to stay metadata-only")
	}
	if !(&MailMessage{Subject: "Factura electrónica 123"}).NeedsFullContent() {
		t.Fatal("expected invoice subject to need full content")
	}
	if !(&MailMessage{
		Subject:     "docs",
		Attachments: []MailAttachmentRef{{Filename: "dian.xml", MimeType: "application/xml"}},
	}).NeedsFullContent() {
		t.Fatal("expected xml attachment to need full content")
	}
}

func TestMailMessageHasFullContent(t *testing.T) {
	t.Parallel()

	if (&MailMessage{Snippet: "hi"}).HasFullContent() {
		t.Fatal("snippet-only metadata is not full content")
	}
	if !(&MailMessage{PlainTextBody: "hello"}).HasFullContent() {
		t.Fatal("expected plain body to count as full content")
	}
}
