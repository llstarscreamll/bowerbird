package application

import (
	"context"
	"strings"

	"github.com/atta/internal/files/api"
	"github.com/atta/internal/files/application/commands"
	appErrors "github.com/atta/internal/platform/errors"
	platformStorage "github.com/atta/internal/platform/storage"
	"github.com/atta/internal/platform/tenant"
)

type tenantObjects struct {
	store platformStorage.FileStore
}

func NewTenantObjects(store platformStorage.FileStore) api.TenantObjects {
	if store == nil {
		panic("file store is required")
	}
	return &tenantObjects{store: store}
}

func (s *tenantObjects) Open(ctx context.Context, module, key string, offset int64) (*api.Object, error) {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return nil, appErrors.New(appErrors.CodeValidation, "tenant id is required")
	}
	module = strings.TrimSpace(module)
	key = strings.TrimSpace(key)
	if module == "" || key == "" || strings.Contains(key, "..") {
		return nil, appErrors.New(appErrors.CodeValidation, "object key is invalid")
	}
	if offset < 0 {
		offset = 0
	}
	prefix := commands.TenantModulePrefix(tenantID, module)
	if !strings.HasPrefix(key, prefix) {
		return nil, appErrors.New(appErrors.CodeValidation, "object was not found")
	}
	opened, err := s.store.OpenFile(ctx, platformStorage.OpenFileInput{Path: key, Offset: offset})
	if err != nil {
		return nil, appErrors.Wrap(err, appErrors.CodeValidation, "object was not found")
	}
	return &api.Object{Body: opened.Body, SizeBytes: opened.SizeBytes}, nil
}
