package domain

import "strings"

// CatalogImportRow is a CSV data line interpreted against catalog item rules.
type CatalogImportRow struct {
	FileRow      int
	InternalCode string
	Name         string
	Kind         string
	Malformed    bool
}

// ImportRowIssue is an immutable classification of why a row cannot become an Item.
type ImportRowIssue struct {
	Column  string
	Code    string
	Message string
}

// Interpret validates a file row. A valid row yields code/name/kind for Item factories.
func (r CatalogImportRow) Interpret() (code InternalCode, name string, kind ItemKind, issue *ImportRowIssue) {
	if r.Malformed {
		return InternalCode{}, "", ItemKind{}, issueOf(ImportErrorMalformedRow, "", r.Kind)
	}
	rawCode := strings.TrimSpace(r.InternalCode)
	if rawCode == "" {
		return InternalCode{}, "", ItemKind{}, issueOf(ImportErrorMissingInternalCode, ImportColumnInternalCode, r.Kind)
	}
	name = strings.TrimSpace(r.Name)
	if name == "" {
		return InternalCode{}, "", ItemKind{}, issueOf(ImportErrorMissingName, ImportColumnName, r.Kind)
	}
	kind, err := ParseImportKind(r.Kind)
	if err != nil {
		return InternalCode{}, "", ItemKind{}, issueOf(ImportErrorInvalidKind, ImportColumnKind, r.Kind)
	}
	code, err = ParseInternalCode(rawCode)
	if err != nil {
		return InternalCode{}, "", ItemKind{}, issueOf(ImportErrorMissingInternalCode, ImportColumnInternalCode, r.Kind)
	}
	return code, name, kind, nil
}

func issueOf(code, column, kindRaw string) *ImportRowIssue {
	return &ImportRowIssue{Column: column, Code: code, Message: ImportErrorMessage(code, kindRaw)}
}
