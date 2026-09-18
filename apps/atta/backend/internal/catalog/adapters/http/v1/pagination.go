package v1

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func pageSize(r *http.Request, fallback, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("page[size]"))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}

func pageAfter(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("page[after]"))
}

func encodeCursor(parts ...string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Join(parts, "\n")))
}

func decodeCursor(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	return strings.Split(string(decoded), "\n")
}

func decodeItemCursor(raw string) (name, id string) {
	parts := decodeCursor(raw)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func decodeImportCursor(raw string) (time.Time, string) {
	parts := decodeCursor(raw)
	if len(parts) != 2 {
		return time.Time{}, ""
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, ""
	}
	return ts, parts[1]
}

func decodeErrorCursor(raw string) (int, string) {
	parts := decodeCursor(raw)
	if len(parts) != 2 {
		return 0, ""
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}
	return n, parts[1]
}

func pageMeta(hasMore bool, cursor string, extra map[string]any) map[string]any {
	meta := map[string]any{"has_more": hasMore}
	if hasMore && cursor != "" {
		meta["cursor"] = cursor
	}
	for k, v := range extra {
		meta[k] = v
	}
	return meta
}
