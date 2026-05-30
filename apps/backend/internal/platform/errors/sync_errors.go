package errors

import "fmt"

const (
	CodeSyncReauthRequired    = "ERR_SYNC_REAUTH_REQUIRED"
	CodeSyncRateLimited       = "ERR_SYNC_RATE_LIMITED"
	CodeSyncProviderTemporary = "ERR_SYNC_PROVIDER_TEMPORARY"
	CodeSyncPayloadRejected   = "ERR_SYNC_PAYLOAD_REJECTED"
	CodeSyncInternal          = "ERR_SYNC_INTERNAL"
)

type SyncErrorOptions struct {
	Provider          string
	AccountEmail      string
	RequiresReauth    bool
	RetryAfterSeconds int
	HelpURL           string
	Meta              map[string]any
}

type SyncError struct {
	Code              string
	Message           string
	Err               error
	Provider          string
	AccountEmail      string
	RequiresReauth    bool
	RetryAfterSeconds int
	HelpURL           string
	Meta              map[string]any
}

func NewSync(code, message string, opts SyncErrorOptions) *SyncError {
	return &SyncError{
		Code:              code,
		Message:           message,
		Provider:          opts.Provider,
		AccountEmail:      opts.AccountEmail,
		RequiresReauth:    opts.RequiresReauth,
		RetryAfterSeconds: opts.RetryAfterSeconds,
		HelpURL:           opts.HelpURL,
		Meta:              cloneMeta(opts.Meta),
	}
}

func WrapSync(err error, code, message string, opts SyncErrorOptions) *SyncError {
	if err == nil {
		return nil
	}

	syncErr := NewSync(code, message, opts)
	syncErr.Err = err
	return syncErr
}

func (e *SyncError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

func (e *SyncError) UXMeta() map[string]any {
	out := cloneMeta(e.Meta)
	if out == nil {
		out = map[string]any{}
	}

	if e.Provider != "" {
		out["provider"] = e.Provider
	}
	if e.AccountEmail != "" {
		out["account_email"] = e.AccountEmail
	}
	if e.RequiresReauth {
		out["requires_reauth"] = true
	}
	if e.RetryAfterSeconds > 0 {
		out["retry_after_seconds"] = e.RetryAfterSeconds
	}

	return out
}

func cloneMeta(meta map[string]any) map[string]any {
	if meta == nil {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		out[k] = v
	}
	return out
}

func HelpURLForCode(code string) string {
	switch code {
	case CodeSyncReauthRequired, CodeSyncRateLimited, CodeSyncProviderTemporary, CodeSyncPayloadRejected, CodeSyncInternal:
		return "https://help.bowerbird.dev/errors/" + code
	default:
		return ""
	}
}
