package queries

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type GetImportByIDQuery struct {
	imports ports.ImportRepository
}

func NewGetImportByIDQuery(imports ports.ImportRepository) *GetImportByIDQuery {
	if imports == nil {
		panic("import repository is required")
	}
	return &GetImportByIDQuery{imports: imports}
}

func (q *GetImportByIDQuery) Execute(ctx context.Context, id string) (*domain.CatalogImport, error) {
	imp, err := q.imports.GetImportByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	return imp, nil
}

type GetActiveImportQuery struct {
	imports ports.ImportRepository
}

func NewGetActiveImportQuery(imports ports.ImportRepository) *GetActiveImportQuery {
	if imports == nil {
		panic("import repository is required")
	}
	return &GetActiveImportQuery{imports: imports}
}

func (q *GetActiveImportQuery) Execute(ctx context.Context) (*domain.CatalogImport, error) {
	return q.imports.GetActiveImport(ctx)
}

type ListImportsQuery struct {
	imports ports.ImportRepository
}

func NewListImportsQuery(imports ports.ImportRepository) *ListImportsQuery {
	if imports == nil {
		panic("import repository is required")
	}
	return &ListImportsQuery{imports: imports}
}

func (q *ListImportsQuery) Execute(ctx context.Context, filter ports.ImportListFilter) (ports.ImportListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}
	return q.imports.ListImports(ctx, filter)
}

type ListImportErrorsQuery struct {
	imports ports.ImportRepository
}

func NewListImportErrorsQuery(imports ports.ImportRepository) *ListImportErrorsQuery {
	if imports == nil {
		panic("import repository is required")
	}
	return &ListImportErrorsQuery{imports: imports}
}

func (q *ListImportErrorsQuery) Execute(ctx context.Context, filter ports.ImportErrorListFilter) (ports.ImportErrorListPage, error) {
	imp, err := q.imports.GetImportByID(ctx, filter.ImportID)
	if err != nil {
		return ports.ImportErrorListPage{}, err
	}
	if imp == nil {
		return ports.ImportErrorListPage{}, appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return q.imports.ListImportErrors(ctx, filter)
}
