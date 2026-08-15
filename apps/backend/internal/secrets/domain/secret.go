package domain

import (
	"strings"
	"time"
)

const (
	PurposeInvoicingDocumentPassword = "invoicing.document_password"
	PurposeIntegrationsAPIKey        = "integrations.api_key"
	PurposeIntegrationsIdPCredential = "integrations.idp_credential"
	PurposeGenericCredential         = "generic.credential"

	KindDocumentPassword = "document_password"
	KindGenericString    = "generic_string"

	KeyIDLocalAESV1 = "local-aes-v1"

	AuditActionCreate = "create"
	AuditActionRotate = "rotate"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
)

type Secret struct {
	ID          string
	Purpose     string
	Kind        string
	Label       string
	Description string
	Ciphertext  []byte
	Version     int
	KeyID       string
	LastUsedAt  *time.Time
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizePurpose(purpose string) string {
	return strings.TrimSpace(purpose)
}

func NormalizeLabel(label string) string {
	return strings.TrimSpace(label)
}

func DefaultKindForPurpose(purpose string) string {
	switch purpose {
	case PurposeInvoicingDocumentPassword:
		return KindDocumentPassword
	default:
		return KindGenericString
	}
}

func IsKnownPurpose(purpose string) bool {
	switch purpose {
	case PurposeInvoicingDocumentPassword,
		PurposeIntegrationsAPIKey,
		PurposeIntegrationsIdPCredential,
		PurposeGenericCredential:
		return true
	default:
		return false
	}
}

type AuditEvent struct {
	ID          string
	SecretID    *string
	Purpose     string
	Action      string
	ActorUserID string
	CreatedAt   time.Time
}

type ResolvedSecret struct {
	ID    string
	Value string
}
