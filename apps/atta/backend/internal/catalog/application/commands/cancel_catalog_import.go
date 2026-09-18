package commands

import (
	"context"
	"errors"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
)

type CancelCatalogImportCommand struct {
	imports ports.ImportRepository
	now     func() time.Time
}

func NewCancelCatalogImportCommand(imports ports.ImportRepository) *CancelCatalogImportCommand {
	if imports == nil {
		panic("import repository is required")
	}
	return &CancelCatalogImportCommand{imports: imports, now: time.Now}
}

func (cmd *CancelCatalogImportCommand) Execute(ctx context.Context, importID string, actor domain.ImportActor) (*domain.CatalogImport, error) {
	imp, err := cmd.imports.GetImportByID(ctx, importID)
	if err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "catalog import not found")
	}
	if err := imp.Cancel(actor, cmd.now().UTC()); err != nil {
		if errors.Is(err, domain.ErrImportNotCancellable) {
			return nil, appErrors.New(appErrors.CodeConflict, "import cannot be cancelled")
		}
		return nil, appErrors.New(appErrors.CodeValidation, err.Error())
	}
	if err := cmd.imports.UpdateImport(ctx, *imp); err != nil {
		return nil, err
	}
	return imp, nil
}
