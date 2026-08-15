package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

// Limiter is a simple per-key fixed window rate limiter for a single API process.
type Limiter struct {
	mu       sync.Mutex
	limit    int
	interval time.Duration
	attempts map[string]attempt
}

type attempt struct {
	count int
	start time.Time
}

func New(limit int, interval time.Duration) *Limiter {
	if limit <= 0 {
		limit = 20
	}
	if interval <= 0 {
		interval = time.Minute
	}
	return &Limiter{
		limit:    limit,
		interval: interval,
		attempts: make(map[string]attempt),
	}
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.attempts[key]
	if !ok || now.Sub(entry.start) >= l.interval {
		l.attempts[key] = attempt{count: 1, start: now}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.attempts[key] = entry
	return true
}

func (l *Limiter) Protect(next api.HandlerFunc) api.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if !l.Allow(clientKey(r)) {
			return appErrors.New(appErrors.CodeRateLimited, "too many requests")
		}
		return next(w, r)
	}
}

func clientKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
