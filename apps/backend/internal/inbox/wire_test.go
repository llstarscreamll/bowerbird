package inbox

import (
	"testing"

	inboxContracts "github.com/bowerbird/internal/inbox/contracts/jobs"
	"github.com/bowerbird/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterSchedulesOmitsRuleWithoutMailOAuth(t *testing.T) {
	assert.Empty(t, RegisterSchedules(config.Config{}))
}

func TestRegisterSchedulesIncludesRuleWhenGmailConfigured(t *testing.T) {
	rules := RegisterSchedules(config.Config{
		GoogleClientID:     "id",
		GoogleClientSecret: "secret",
	})
	require.Len(t, rules, 1)
	assert.Equal(t, "inbox-sync-all", rules[0].Name)
	assert.Equal(t, inboxContracts.InboxSyncAllAccountsType, rules[0].JobType)
}

func TestRegisterSchedulesIncludesRuleWhenMicrosoftConfigured(t *testing.T) {
	rules := RegisterSchedules(config.Config{
		MicrosoftClientID:     "id",
		MicrosoftClientSecret: "secret",
	})
	require.Len(t, rules, 1)
	assert.Equal(t, inboxContracts.InboxSyncAllAccountsType, rules[0].JobType)
}
