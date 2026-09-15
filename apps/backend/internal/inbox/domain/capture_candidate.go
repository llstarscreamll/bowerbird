package domain

import (
	"path/filepath"
	"strings"
)

func (m *MailMessage) NeedsFullContent() bool {
	if m == nil {
		return false
	}
	if hasInvoiceKeyword(m.Subject, m.Snippet, m.PlainTextBody) {
		return true
	}
	return hasInvoiceAttachment(m.Attachments)
}

func hasInvoiceKeyword(values ...string) bool {
	combined := strings.ToLower(strings.TrimSpace(strings.Join(values, "\n")))
	if combined == "" {
		return false
	}
	for _, keyword := range []string{
		"factura electronica",
		"facturación electrónica",
		"factura electrónica",
		"facturacion electronica",
		"facturacion",
		"factura",
		"invoice",
	} {
		if strings.Contains(combined, keyword) {
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
