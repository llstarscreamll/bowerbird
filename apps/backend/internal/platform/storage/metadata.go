package storage

import (
	"strings"
	"unicode"
)

// SanitizeObjectMetadataValue normalizes a value for S3 user metadata (US-ASCII).
// Applied for AWS S3 and MinIO alike: HTTP header constraints; MinIO signing is stricter for some Unicode (e.g. U+202F).
func SanitizeObjectMetadataValue(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "unknown"
	}

	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		switch {
		case r >= 32 && r <= 126:
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		default:
			b.WriteRune('_')
		}
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "unknown"
	}
	if len(out) > 256 {
		return out[:256]
	}
	return out
}

// SanitizeObjectMetadata returns a copy with all values normalized for S3 user metadata.
func SanitizeObjectMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return metadata
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = SanitizeObjectMetadataValue(value)
	}
	return out
}

// MetadataToPresignHeaders maps sanitized S3 user metadata to HTTP headers the client must send with a presigned PUT.
func MetadataToPresignHeaders(metadata map[string]string) map[string]string {
	sanitized := SanitizeObjectMetadata(metadata)
	if len(sanitized) == 0 {
		return nil
	}
	out := make(map[string]string, len(sanitized))
	for key, value := range sanitized {
		out["x-amz-meta-"+key] = value
	}
	return out
}
