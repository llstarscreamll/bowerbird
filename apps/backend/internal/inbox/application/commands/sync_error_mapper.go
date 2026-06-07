package commands

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	connectionsApp "github.com/bowerbird/internal/connections/application"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

var errPayloadRejected = errors.New("sync payload rejected")

var statusCodePattern = regexp.MustCompile(`(?i)status\s+(\d{3})`)
var retryAfterPattern = regexp.MustCompile(`(?i)retry-after\s*=\s*"?([0-9]+)"?`)

func classifySyncError(account connectionsApp.ConnectionInfo, err error) error {
	if err == nil {
		return nil
	}

	var existingSyncErr *appErrors.SyncError
	if errors.As(err, &existingSyncErr) {
		return err
	}

	providerName := normalizeProviderForDetail(account.Provider)
	accountEmail := strings.TrimSpace(account.ProviderAccountEmail)
	detail := fmt.Sprintf("La cuenta de %s %s requiere atención.", providerName, accountEmail)

	if errors.Is(err, errPayloadRejected) {
		return appErrors.WrapSync(err, appErrors.CodeSyncPayloadRejected, detail, appErrors.SyncErrorOptions{
			Provider:     normalizeProviderForMeta(account.Provider),
			AccountEmail: accountEmail,
		})
	}

	errText := strings.ToLower(err.Error())
	statusCode := parseStatusCode(errText)
	retryAfterSeconds := parseRetryAfterSeconds(errText)

	if isReauthError(statusCode, errText) {
		return appErrors.WrapSync(err, appErrors.CodeSyncReauthRequired, detail, appErrors.SyncErrorOptions{
			Provider:       normalizeProviderForMeta(account.Provider),
			AccountEmail:   accountEmail,
			RequiresReauth: true,
		})
	}

	if isRateLimitedError(statusCode, errText) {
		if retryAfterSeconds == 0 {
			retryAfterSeconds = 120
		}
		return appErrors.WrapSync(err, appErrors.CodeSyncRateLimited, detail, appErrors.SyncErrorOptions{
			Provider:          normalizeProviderForMeta(account.Provider),
			AccountEmail:      accountEmail,
			RetryAfterSeconds: retryAfterSeconds,
		})
	}

	if isProviderTemporaryError(statusCode, errText) {
		return appErrors.WrapSync(err, appErrors.CodeSyncProviderTemporary, detail, appErrors.SyncErrorOptions{
			Provider:     normalizeProviderForMeta(account.Provider),
			AccountEmail: accountEmail,
		})
	}

	return appErrors.WrapSync(err, appErrors.CodeSyncInternal, detail, appErrors.SyncErrorOptions{
		Provider:     normalizeProviderForMeta(account.Provider),
		AccountEmail: accountEmail,
	})
}

func shouldMarkRequiresReconnect(err error) bool {
	var syncErr *appErrors.SyncError
	if !errors.As(err, &syncErr) {
		return false
	}

	return syncErr.Code == appErrors.CodeSyncReauthRequired
}

func syncErrorCode(err error) string {
	var syncErr *appErrors.SyncError
	if errors.As(err, &syncErr) {
		return syncErr.Code
	}

	return appErrors.CodeSyncInternal
}

func parseStatusCode(errText string) int {
	matches := statusCodePattern.FindStringSubmatch(errText)
	if len(matches) != 2 {
		return 0
	}

	statusCode, convErr := strconv.Atoi(matches[1])
	if convErr != nil {
		return 0
	}

	return statusCode
}

func parseRetryAfterSeconds(errText string) int {
	matches := retryAfterPattern.FindStringSubmatch(errText)
	if len(matches) != 2 {
		return 0
	}

	retryAfterSeconds, convErr := strconv.Atoi(matches[1])
	if convErr != nil || retryAfterSeconds <= 0 {
		return 0
	}

	return retryAfterSeconds
}

func isReauthError(statusCode int, errText string) bool {
	if statusCode == 401 || statusCode == 403 {
		return true
	}

	return strings.Contains(errText, "invalid_grant") ||
		strings.Contains(errText, "token expired") ||
		strings.Contains(errText, "token has been expired") ||
		strings.Contains(errText, "token revoked") ||
		strings.Contains(errText, "invalid credentials") ||
		strings.Contains(errText, "reauth")
}

func isRateLimitedError(statusCode int, errText string) bool {
	if statusCode == 429 {
		return true
	}

	return strings.Contains(errText, "rate limit") ||
		strings.Contains(errText, "too many requests") ||
		strings.Contains(errText, "quota exceeded")
}

func isProviderTemporaryError(statusCode int, errText string) bool {
	if statusCode == 502 || statusCode == 503 || statusCode == 504 {
		return true
	}

	return strings.Contains(errText, "timeout") ||
		strings.Contains(errText, "temporarily unavailable") ||
		strings.Contains(errText, "service unavailable")
}

func normalizeProviderForMeta(provider string) string {
	normalized := strings.ToUpper(strings.TrimSpace(provider))
	if normalized == "" {
		return "UNKNOWN"
	}
	return normalized
}

func normalizeProviderForDetail(provider string) string {
	normalized := strings.TrimSpace(provider)
	if normalized == "" {
		return "correo"
	}

	switch strings.ToUpper(normalized) {
	case "GMAIL":
		return "Gmail"
	case "OUTLOOK":
		return "Outlook"
	case "HOTMAIL":
		return "Hotmail"
	case "YAHOO":
		return "Yahoo"
	case "MICROSOFT":
		return "Microsoft"
	default:
		return normalized
	}
}
