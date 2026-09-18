package catalog

import (
	"testing"

	contractJobs "github.com/atta/internal/catalog/contracts/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterSchedules(t *testing.T) {
	rules := RegisterSchedules()
	require.Len(t, rules, 1)
	assert.Equal(t, "catalog-import-purge", rules[0].Name)
	assert.Equal(t, "0 5 * * *", rules[0].Schedule)
	assert.Equal(t, contractJobs.CatalogImportPurgeRequestedType, rules[0].JobType)
}
