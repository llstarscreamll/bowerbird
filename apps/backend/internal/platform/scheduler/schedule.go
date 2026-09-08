package scheduler

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type Schedule interface {
	Next(from time.Time) time.Time
}

func ParseSchedule(expr string) (Schedule, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return nil, fmt.Errorf("schedule is required")
	}
	if strings.HasPrefix(raw, "cron(") || strings.Contains(raw, "?") {
		return nil, fmt.Errorf("unsupported schedule %q: use EventBridge rate() or Unix crontab (5 fields, UTC)", raw)
	}
	if strings.HasPrefix(raw, "rate(") {
		return parseRate(raw)
	}
	return parseUnixCron(raw)
}

type rateSchedule struct {
	d time.Duration
}

func (r rateSchedule) Next(from time.Time) time.Time {
	return from.Add(r.d)
}

func parseRate(expr string) (Schedule, error) {
	if !strings.HasSuffix(expr, ")") {
		return nil, fmt.Errorf("invalid rate expression %q", expr)
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expr, "rate("), ")"))
	parts := strings.Split(inner, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rate expression %q", expr)
	}

	var n int
	if _, err := fmt.Sscanf(parts[0], "%d", &n); err != nil || n < 1 || parts[0] != fmt.Sprintf("%d", n) {
		return nil, fmt.Errorf("invalid rate value in %q", expr)
	}

	unit := parts[1]
	singular := n == 1
	var d time.Duration
	switch unit {
	case "minute":
		if !singular {
			return nil, fmt.Errorf("invalid rate expression %q: use minutes", expr)
		}
		d = time.Minute
	case "minutes":
		if singular {
			return nil, fmt.Errorf("invalid rate expression %q: use minute", expr)
		}
		d = time.Duration(n) * time.Minute
	case "hour":
		if !singular {
			return nil, fmt.Errorf("invalid rate expression %q: use hours", expr)
		}
		d = time.Hour
	case "hours":
		if singular {
			return nil, fmt.Errorf("invalid rate expression %q: use hour", expr)
		}
		d = time.Duration(n) * time.Hour
	case "day":
		if !singular {
			return nil, fmt.Errorf("invalid rate expression %q: use days", expr)
		}
		d = 24 * time.Hour
	case "days":
		if singular {
			return nil, fmt.Errorf("invalid rate expression %q: use day", expr)
		}
		d = time.Duration(n) * 24 * time.Hour
	default:
		return nil, fmt.Errorf("invalid rate unit in %q", expr)
	}
	return rateSchedule{d: d}, nil
}

type cronSchedule struct {
	inner cron.Schedule
}

func (c cronSchedule) Next(from time.Time) time.Time {
	return c.inner.Next(from.UTC())
}

func parseUnixCron(expr string) (Schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("unix crontab must have 5 fields, got %d in %q", len(fields), expr)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid crontab %q: %w", expr, err)
	}
	return cronSchedule{inner: sched}, nil
}
