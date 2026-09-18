package domain

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"unicode"
)

func (m *MailMessage) NeedsFullContent() bool {
	if m == nil {
		return false
	}
	if LooksLikeInvoiceCapture(m.Subject, m.Snippet, m.PlainTextBody, m.HTMLBody, m.Sender) {
		return true
	}
	return hasInvoiceAttachment(m.Attachments)
}

func LooksLikeInvoiceCapture(values ...string) bool {
	combined := foldInvoiceText(strings.Join(values, "\n"))
	if combined == "" {
		return false
	}
	if hasInvoiceKeyword(combined) {
		return true
	}
	if hasDIANInvoiceSubject(combined) {
		return true
	}
	return hasInvoiceProviderSender(combined)
}

func hasInvoiceKeyword(combined string) bool {
	for _, keyword := range []string{
		"factura electronica",
		"facturacion electronica",
		"facturacion",
		"factura",
		"invoice",
		"documento electronico",
		"fichero .zip",
		"archivo .zip",
		"nomina electronica",
	} {
		if strings.Contains(combined, keyword) {
			return true
		}
	}
	return false
}

func hasDIANInvoiceSubject(combined string) bool {
	for _, line := range strings.Split(combined, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 4 {
			continue
		}
		nit := digitsOnly(parts[0])
		if len(nit) < 8 || len(nit) > 12 {
			continue
		}
		switch strings.TrimSpace(parts[3]) {
		case "01", "02", "03", "04", "05", "20", "91", "92", "94", "95", "96":
			return true
		}
	}
	return false
}

func hasInvoiceProviderSender(combined string) bool {
	for _, token := range []string{
		"saphety",
		"paperless",
		"carvajal",
		"facelec",
		"dte.",
	} {
		if strings.Contains(combined, token) {
			return true
		}
	}
	return false
}

func hasInvoiceAttachment(refs []MailAttachmentRef) bool {
	for _, ref := range refs {
		ext := strings.ToLower(filepath.Ext(ref.Filename))
		if ext == ".xml" || ext == ".pdf" || ext == ".zip" {
			return true
		}
		mime := strings.ToLower(strings.TrimSpace(ref.MimeType))
		if strings.Contains(mime, "pdf") || strings.Contains(mime, "xml") || strings.Contains(mime, "zip") {
			return true
		}
	}
	return false
}

func (m *InboxMessage) CaptureHint() *MailMessage {
	if m == nil {
		return nil
	}
	hint := MailMessage{}
	if len(m.rawData) > 0 {
		_ = json.Unmarshal(m.rawData, &hint)
	}
	if hint.Subject == "" && m.subject != nil {
		hint.Subject = *m.subject
	}
	if hint.Snippet == "" && m.snippet != nil {
		hint.Snippet = *m.snippet
	}
	if hint.Sender == "" && m.senderEmail != nil {
		hint.Sender = *m.senderEmail
	}
	return &hint
}

func foldInvoiceText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch r {
		case 'á', 'à', 'ä', 'â':
			b.WriteByte('a')
		case 'é', 'è', 'ë', 'ê':
			b.WriteByte('e')
		case 'í', 'ì', 'ï', 'î':
			b.WriteByte('i')
		case 'ó', 'ò', 'ö', 'ô':
			b.WriteByte('o')
		case 'ú', 'ù', 'ü', 'û':
			b.WriteByte('u')
		case 'ñ':
			b.WriteByte('n')
		default:
			if unicode.IsSpace(r) {
				b.WriteByte(' ')
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func digitsOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
