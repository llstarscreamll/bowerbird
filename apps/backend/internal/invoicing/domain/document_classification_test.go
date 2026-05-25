package domain

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestInvoiceDocumentClassifierClassifyAttachmentsGroupsXMLAndPDF(t *testing.T) {
	classifier := NewInvoiceDocumentClassifier()

	result, err := classifier.ClassifyAttachments([]AttachmentContent{
		{Filename: "FE-100.xml", S3Key: "k1", Data: []byte("<Invoice></Invoice>")},
		{Filename: "FE-100.pdf", S3Key: "k2", Data: []byte("%PDF-1.7")},
	})
	if err != nil {
		t.Fatalf("classify attachments: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].XML == nil || result.Groups[0].PDF == nil {
		t.Fatalf("expected xml and pdf in same group")
	}
}

func TestInvoiceDocumentClassifierClassifyAttachmentsExpandsZIP(t *testing.T) {
	classifier := NewInvoiceDocumentClassifier()
	zipBytes := makeTestZip(t, map[string][]byte{
		"FE-200.xml": []byte("<Invoice></Invoice>"),
		"FE-200.pdf": []byte("%PDF-1.7"),
	})

	result, err := classifier.ClassifyAttachments([]AttachmentContent{
		{Filename: "bundle.zip", S3Key: "zip-key", Data: zipBytes},
	})
	if err != nil {
		t.Fatalf("classify attachments: %v", err)
	}

	if result.CompressedSources != 1 {
		t.Fatalf("expected 1 compressed source, got %d", result.CompressedSources)
	}
	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group after unzip, got %d", len(result.Groups))
	}
}

func makeTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip file: %v", err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write zip file: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buf.Bytes()
}
