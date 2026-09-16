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

func TestMailMessageNeedsFullContentSaphetyDIAN(t *testing.T) {
	t.Parallel()

	msg := &MailMessage{
		Subject: "860512780;universidad nacional abierta y a distancia;FVRC2573887;01;universidad nacional abierta y a distancia",
		Sender:  "noreply@saphety.com.co",
		Snippet: "Estimados Señores, La empresa UNIVERSIDAD NACIONAL ABIERTA YA DISTANCIA le ha emitido el documento electrónico abajo indicado, el cual se encuentra adjunto a este correo, en un fichero .zip.",
	}
	if !msg.NeedsFullContent() {
		t.Fatal("expected saphety DIAN email to need full content from metadata")
	}
}

func TestMailMessageNeedsFullContentPaperlessSubject(t *testing.T) {
	t.Parallel()

	msg := &MailMessage{
		Subject: "900277370; I SHOP COLOMBIA SAS; FETA19245; 01; I SHOP COLOMBIA SAS",
		Sender:  "Documento Electronico ISHOP <dte_9002773704@dte.paperless.com.co>",
	}
	if !msg.NeedsFullContent() {
		t.Fatal("expected paperless DIAN email to need full content from metadata")
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

func TestInboxMessageCaptureHintUsesStoredMetadata(t *testing.T) {
	t.Parallel()

	subject := "860512780;unad;FVRC2573887;01;unad"
	snippet := "documento electrónico en un fichero .zip"
	sender := "noreply@saphety.com.co"
	message, err := NewInboxMessageAsSynced(NewInboxMessageInput{
		ID:                "msg-1",
		ConnectionID:      "acc-1",
		ProviderMessageID: "prov-1",
		Subject:           &subject,
		SenderEmail:       &sender,
		Snippet:           &snippet,
	})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	hint := message.CaptureHint()
	if hint == nil || !hint.NeedsFullContent() {
		t.Fatal("expected stored saphety stub to look like an invoice capture")
	}
}
