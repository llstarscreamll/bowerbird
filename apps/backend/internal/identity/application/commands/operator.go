package commands

import (
	"context"
	"strings"

	"github.com/bowerbird/internal/identity/application/ports"
	"github.com/bowerbird/internal/identity/domain"
)

func EnsureOperatorRole(ctx context.Context, repo ports.Repository, operatorEmails []string, user *domain.User) error {
	if user == nil || user.IsPlatformOperator() {
		return nil
	}
	if !isOperatorEmail(user.Email, operatorEmails) {
		return nil
	}
	user.GrantPlatformOperator()
	return repo.SetPlatformRole(ctx, user.ID, domain.PlatformRoleOperator)
}

func isOperatorEmail(email string, operatorEmails []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, allowed := range operatorEmails {
		if normalized == allowed {
			return true
		}
	}
	return false
}
