package attestation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

const HeaderKey = "tenant_attestation"

var (
	ErrMissingAttestation = errors.New("missing tenant attestation")
	ErrInvalidAttestation = errors.New("invalid tenant attestation")
)

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

func Sign(secret, messageID, tenantSlug, kind string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(messageID + "|" + tenantSlug + "|" + kind))
	return hex.EncodeToString(mac.Sum(nil))
}

func (v *Verifier) Sign(messageID, tenantSlug, kind string) string {
	return Sign(string(v.secret), messageID, tenantSlug, kind)
}

func (v *Verifier) Verify(messageID, tenantSlug, kind, signature string) error {
	if signature == "" {
		return ErrMissingAttestation
	}
	expected := v.Sign(messageID, tenantSlug, kind)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("%w for message %q", ErrInvalidAttestation, messageID)
	}
	return nil
}
