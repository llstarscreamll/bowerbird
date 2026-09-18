package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScheduleRate(t *testing.T) {
	from := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	sched, err := ParseSchedule("rate(1 hour)")
	require.NoError(t, err)
	assert.Equal(t, from.Add(time.Hour), sched.Next(from))

	sched, err = ParseSchedule("rate(5 minutes)")
	require.NoError(t, err)
	assert.Equal(t, from.Add(5*time.Minute), sched.Next(from))

	sched, err = ParseSchedule("rate(2 days)")
	require.NoError(t, err)
	assert.Equal(t, from.Add(48*time.Hour), sched.Next(from))
}

func TestParseScheduleUnixCron(t *testing.T) {
	from := time.Date(2026, 9, 7, 12, 30, 0, 0, time.UTC)
	sched, err := ParseSchedule("0 * * * *")
	require.NoError(t, err)
	next := sched.Next(from)
	assert.Equal(t, time.Date(2026, 9, 7, 13, 0, 0, 0, time.UTC), next)
}

func TestParseScheduleRejectsInvalid(t *testing.T) {
	cases := []string{
		"rate(1 hours)",
		"rate(5 minute)",
		"cron(0 * * * ? *)",
		"cron(0 * * * *)",
		"0 * * * ? *",
		"0 * * * * *",
		"",
	}
	for _, expr := range cases {
		_, err := ParseSchedule(expr)
		assert.Error(t, err, expr)
	}
}
