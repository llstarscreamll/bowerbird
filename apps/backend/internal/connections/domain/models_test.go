package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectionRequiresIdentity(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	_, err := NewConnection("", "user-1", "gmail", "a@b.com", nil, SharingPolicyPrivate, now)
	assert.ErrorIs(t, err, ErrConnectionIDRequired)

	_, err = NewConnection("c-1", "", "gmail", "a@b.com", nil, SharingPolicyPrivate, now)
	assert.ErrorIs(t, err, ErrOwnerRequired)

	_, err = NewConnection("c-1", "user-1", "", "a@b.com", nil, SharingPolicyPrivate, now)
	assert.ErrorIs(t, err, ErrProviderRequired)

	_, err = NewConnection("c-1", "user-1", "gmail", "", nil, SharingPolicyPrivate, now)
	assert.ErrorIs(t, err, ErrProviderAccountEmailRequired)

	_, err = NewConnection("c-1", "user-1", "gmail", "a@b.com", nil, "public", now)
	assert.ErrorIs(t, err, ErrInvalidSharingPolicy)
}

func TestNewConnectionStartsActiveAndPrivateByDefault(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	conn, err := NewConnection("c-1", "user-1", "gmail", "a@b.com", []string{"mail.read"}, "", now)
	require.NoError(t, err)
	assert.Equal(t, ConnectionStatusActive, conn.Status)
	assert.Equal(t, SharingPolicyPrivate, conn.SharingPolicy)
	assert.Equal(t, []string{"mail.read"}, conn.GrantedScopes)
}

func TestSealCredentials(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	conn, err := NewConnection("c-1", "user-1", "gmail", "a@b.com", nil, SharingPolicyPrivate, now)
	require.NoError(t, err)

	assert.ErrorIs(t, conn.SealCredentials(nil, now), ErrCredentialsRequired)

	sealedAt := now.Add(time.Minute)
	require.NoError(t, conn.SealCredentials([]byte("cipher"), sealedAt))
	assert.Equal(t, []byte("cipher"), conn.EncryptedCredentials)
	assert.Equal(t, sealedAt, conn.UpdatedAt)
}

func TestConnectionVisibleToRespectsSharingPolicy(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	private, err := NewConnection("c-1", "user-1", "gmail", "a@b.com", nil, SharingPolicyPrivate, now)
	require.NoError(t, err)
	assert.True(t, private.IsPrivate())
	assert.True(t, private.VisibleTo("user-1"))
	assert.False(t, private.VisibleTo("user-2"))

	shared, err := NewConnection("c-2", "user-1", "gmail", "a@b.com", nil, SharingPolicyTenantAll, now)
	require.NoError(t, err)
	assert.False(t, shared.IsPrivate())
	assert.True(t, shared.VisibleTo("user-2"))
}

func TestMarkRequiresReconnectRequiresReason(t *testing.T) {
	conn, err := NewConnection("c-1", "user-1", "gmail", "a@b.com", nil, SharingPolicyPrivate, time.Now())
	require.NoError(t, err)

	assert.ErrorIs(t, conn.MarkRequiresReconnect(" ", time.Now()), ErrReconnectReasonRequired)
	require.NoError(t, conn.MarkRequiresReconnect("token expired", time.Now()))
	assert.Equal(t, ConnectionStatusRequiresReconnect, conn.Status)
}
