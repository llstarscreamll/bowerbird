package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntitlementIsEffective(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	ends := now.Add(24 * time.Hour)
	expired := now.Add(-time.Hour)

	tests := []struct {
		name string
		ent  Entitlement
		want bool
	}{
		{
			name: "active open-ended",
			ent:  Entitlement{Status: StatusActive, StartsAt: now.Add(-time.Hour)},
			want: true,
		},
		{
			name: "trial with future end",
			ent:  Entitlement{Status: StatusTrial, StartsAt: now.Add(-time.Hour), EndsAt: &ends},
			want: true,
		},
		{
			name: "trial expired",
			ent:  Entitlement{Status: StatusTrial, StartsAt: now.Add(-48 * time.Hour), EndsAt: &expired},
			want: false,
		},
		{
			name: "not yet started",
			ent:  Entitlement{Status: StatusActive, StartsAt: now.Add(time.Hour)},
			want: false,
		},
		{
			name: "suspended",
			ent:  Entitlement{Status: StatusSuspended, StartsAt: now.Add(-time.Hour)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ent.IsEffective(now))
		})
	}
}

func TestDefaultPackExcludesSend(t *testing.T) {
	keys := DefaultPackFeatureKeys()
	assert.Contains(t, keys, FeatureMailInbox)
	assert.Contains(t, keys, FeatureInvoicingCaptureFromEmail)
	assert.NotContains(t, keys, FeatureMailSend)
}
