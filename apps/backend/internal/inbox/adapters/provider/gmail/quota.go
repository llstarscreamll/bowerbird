package gmail

import (
	"context"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	gmailUnitsPerMinute     = 6000
	gmailUnitBudgetHeadroom = gmailUnitsPerMinute * 4 / 5
	gmailGetCost            = 20
	gmailAttachmentCost     = 20
	gmailListCost           = 5
	gmailHistoryCost        = 2
	gmailProfileCost        = 1
	gmailModifyCost         = 5
	gmailLabelCost          = 5
	gmailTrashCost          = 20
	gmailSendCost           = 100
	gmailMaxRetries         = 5
)

var (
	gmailBackoffBase      = time.Second
	gmailBackoffMax       = 32 * time.Second
	gmailBackoffJitterMax = time.Second
	gmailQuotaMu          sync.Mutex
	gmailQuotaBuckets     = map[string]*gmailUserQuota{}
)

type gmailUserQuota struct {
	mu   sync.Mutex
	next time.Time
}

func gmailPace(units int) time.Duration {
	if units <= 0 {
		units = 1
	}
	return time.Duration(units) * time.Minute / gmailUnitBudgetHeadroom
}

func waitGmailUnits(ctx context.Context, key string, units int) error {
	if key == "" {
		key = "default"
	}
	interval := gmailPace(units)

	gmailQuotaMu.Lock()
	bucket := gmailQuotaBuckets[key]
	if bucket == nil {
		bucket = &gmailUserQuota{}
		gmailQuotaBuckets[key] = bucket
	}
	gmailQuotaMu.Unlock()

	bucket.mu.Lock()
	now := time.Now()
	waitUntil := bucket.next
	if waitUntil.Before(now) {
		waitUntil = now
	}
	bucket.next = waitUntil.Add(interval)
	bucket.mu.Unlock()

	delay := time.Until(waitUntil)
	if delay <= 0 {
		return nil
	}
	if delay >= time.Second {
		slog.Info("inbox.gmail.quota paced", "units", units, "wait_ms", delay.Milliseconds())
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func gmailBackoffDelay(attempt int, retryAfter string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	delay := gmailBackoffBase << attempt
	if delay > gmailBackoffMax {
		delay = gmailBackoffMax
	}
	if gmailBackoffJitterMax > 0 {
		delay += time.Duration(rand.IntN(int(gmailBackoffJitterMax)))
	}
	return delay
}

func isGmailRateLimitResponse(resp *http.Response, body string) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return false
	}
	lower := strings.ToLower(body)
	return strings.Contains(lower, "ratelimitexceeded") ||
		strings.Contains(lower, "rate_limit_exceeded") ||
		strings.Contains(lower, "quota exceeded") ||
		strings.Contains(lower, "rate limit")
}

func peekRateLimitBody(resp *http.Response) (string, io.ReadCloser, error) {
	if resp == nil || resp.Body == nil {
		return "", http.NoBody, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
	if err != nil {
		return "", http.NoBody, err
	}
	return string(raw), io.NopCloser(strings.NewReader(string(raw))), nil
}
