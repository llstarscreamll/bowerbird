package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionInfoVisibleTo(t *testing.T) {
	private := ConnectionInfo{OwnerUserID: "user-1", SharingPolicy: SharingPolicyPrivate}
	assert.True(t, private.VisibleTo("user-1"))
	assert.False(t, private.VisibleTo("user-2"))

	shared := ConnectionInfo{OwnerUserID: "user-1", SharingPolicy: "tenant_all"}
	assert.True(t, shared.VisibleTo("user-2"))
}
