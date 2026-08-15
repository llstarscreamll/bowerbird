package application

import (
	"context"
	"fmt"
	"slices"

	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/rbac/application/ports"
)

type Service struct {
	repo ports.PermissionRepository
}

func NewService(repo ports.PermissionRepository) *Service {
	if repo == nil {
		panic("rbac permission repository is required")
	}
	return &Service{repo: repo}
}

func (s *Service) ListEffectivePermissions(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}
	codes, err := s.repo.ListCodesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return codes, nil
}

func (s *Service) HasPermission(ctx context.Context, userID, code string) (bool, error) {
	codes, err := s.ListEffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return slices.Contains(codes, code), nil
}

func (s *Service) RequirePermission(ctx context.Context, code string) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	ok, err := s.HasPermission(ctx, claims.UserID, code)
	if err != nil {
		return appErrors.Wrap(err, appErrors.CodeInternal, "failed to check permission")
	}
	if !ok {
		return appErrors.New(appErrors.CodeForbidden, "permission required: "+code)
	}
	return nil
}
