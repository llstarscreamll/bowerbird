package application_test

import (
	"context"
	"testing"

	"github.com/bowerbird/internal/rbac/application"
	rbacDomain "github.com/bowerbird/internal/rbac/domain"
	"github.com/stretchr/testify/require"
)

type stubPermissionRepo struct {
	codes map[string][]string
}

func (s *stubPermissionRepo) ListCodesForUser(ctx context.Context, userID string) ([]string, error) {
	return s.codes[userID], nil
}

func TestHasPermission(t *testing.T) {
	svc := application.NewService(&stubPermissionRepo{codes: map[string][]string{
		"user-1": {rbacDomain.PermissionSecretsRead},
	}})

	ok, err := svc.HasPermission(context.Background(), "user-1", rbacDomain.PermissionSecretsRead)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = svc.HasPermission(context.Background(), "user-1", rbacDomain.PermissionSecretsWrite)
	require.NoError(t, err)
	require.False(t, ok)
}
