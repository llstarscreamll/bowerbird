package domain

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"sort"
	"strings"
)

type DocumentKind string

const (
	DocumentKindZIP   DocumentKind = "zip"
	DocumentKindXML   DocumentKind = "xml"
	DocumentKindPDF   DocumentKind = "pdf"
	DocumentKindOther DocumentKind = "other"
)

type AttachmentContent struct {
	Filename string
	S3Key    string
	Data     []byte
}

type ClassifiedDocument struct {
	Filename      string
	S3Key         string
	Source        string
	ParentArchive string
	Kind          DocumentKind
	Data          []byte
}

type DocumentGroup struct {
	GroupKey  string
	XML       *ClassifiedDocument
	PDF       *ClassifiedDocument
	Documents []ClassifiedDocument
}

type ClassificationResult struct {
	Groups            []DocumentGroup
	Unclassified      []ClassifiedDocument
	TotalDocuments    int
	CompressedSources int
}

type DocumentClassifier interface {
	ClassifyAttachments(attachments []AttachmentContent) (*ClassificationResult, error)
}

type InvoiceDocumentClassifier struct{}

func NewInvoiceDocumentClassifier() *InvoiceDocumentClassifier {
	return &InvoiceDocumentClassifier{}
}

func (c *InvoiceDocumentClassifier) ClassifyAttachments(attachments []AttachmentContent) (*ClassificationResult, error) {
	documents := make([]ClassifiedDocument, 0, len(attachments))
	compressedSources := 0

	for _, attachment := range attachments {
		kind := detectDocumentKind(attachment.Filename, attachment.Data)
		document := ClassifiedDocument{
			Filename: attachment.Filename,
			S3Key:    attachment.S3Key,
			Source:   "s3",
			Kind:     kind,
			Data:     attachment.Data,
		}

		if kind != DocumentKindZIP {
			documents = append(documents, document)
			continue
		}

		compressedSources++
		extracted, err := decompressZIP(document)
		if err != nil {
			return nil, err
		}
		documents = append(documents, extracted...)
	}

	groupsMap := map[string]*DocumentGroup{}
	result := &ClassificationResult{
		Unclassified:      make([]ClassifiedDocument, 0),
		CompressedSources: compressedSources,
	}

	for _, doc := range documents {
		result.TotalDocuments++
		if doc.Kind != DocumentKindXML && doc.Kind != DocumentKindPDF {
			result.Unclassified = append(result.Unclassified, doc)
			continue
		}

		groupKey := normalizeGroupKey(doc.Filename)
		if groupKey == "" {
			result.Unclassified = append(result.Unclassified, doc)
			continue
		}

		group, ok := groupsMap[groupKey]
		if !ok {
			group = &DocumentGroup{GroupKey: groupKey, Documents: make([]ClassifiedDocument, 0, 2)}
			groupsMap[groupKey] = group
		}

		group.Documents = append(group.Documents, doc)
		if doc.Kind == DocumentKindXML && group.XML == nil {
			copyDoc := doc
			group.XML = &copyDoc
		}
		if doc.Kind == DocumentKindPDF && group.PDF == nil {
			copyDoc := doc
			group.PDF = &copyDoc
		}
	}

	groupKeys := make([]string, 0, len(groupsMap))
	for key := range groupsMap {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	result.Groups = make([]DocumentGroup, 0, len(groupKeys))
	for _, key := range groupKeys {
		result.Groups = append(result.Groups, *groupsMap[key])
	}

	return result, nil
}

func (g DocumentGroup) SupportsInvoiceExtraction() bool {
	return g.XML != nil || g.PDF != nil
}

func (g DocumentGroup) PreferredDocumentSource() string {
	if g.XML != nil {
		return "xml"
	}
	if g.PDF != nil {
		return "llm"
	}

	return ""
}

func decompressZIP(source ClassifiedDocument) ([]ClassifiedDocument, error) {
	reader, err := zip.NewReader(bytes.NewReader(source.Data), int64(len(source.Data)))
	if err != nil {
		return nil, err
	}

	files := make([]ClassifiedDocument, 0, len(reader.File))
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		contentReader, err := f.Open()
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(contentReader); err != nil {
			_ = contentReader.Close()
			return nil, err
		}
		_ = contentReader.Close()

		name := filepath.Base(f.Name)
		kind := detectDocumentKind(name, buf.Bytes())
		files = append(files, ClassifiedDocument{
			Filename:      name,
			S3Key:         source.S3Key,
			Source:        "zip",
			ParentArchive: source.Filename,
			Kind:          kind,
			Data:          buf.Bytes(),
		})
	}

	return files, nil
}

func detectDocumentKind(filename string, data []byte) DocumentKind {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return DocumentKindZIP
	case ".xml":
		return DocumentKindXML
	case ".pdf":
		return DocumentKindPDF
	}

	if len(data) >= 4 {
		if data[0] == 'P' && data[1] == 'K' {
			return DocumentKindZIP
		}
		if bytes.HasPrefix(data, []byte("%PDF")) {
			return DocumentKindPDF
		}
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		return DocumentKindXML
	}

	return DocumentKindOther
}

func normalizeGroupKey(filename string) string {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(filename)))
	if base == "" {
		return ""
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "")
	return replacer.Replace(stem)
}
