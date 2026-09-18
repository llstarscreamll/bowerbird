package commands

import (
	"context"
	"errors"
	"time"

	"github.com/atta/internal/catalog/application/ports"
	contractJobs "github.com/atta/internal/catalog/contracts/jobs"
	"github.com/atta/internal/catalog/domain"
	filesapi "github.com/atta/internal/files/api"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/atta/internal/platform/id"
	"github.com/atta/internal/platform/jobs"
)

type QueueCatalogImportInput struct {
	ID        string
	FileKey   string
	Requester domain.ImportActor
}

type QueueCatalogImportCommand struct {
	imports  ports.ImportRepository
	files    filesapi.TenantObjects
	jobQueue jobs.TaskQueue
	now      func() time.Time
}

func NewQueueCatalogImportCommand(imports ports.ImportRepository, files filesapi.TenantObjects, jobQueue jobs.TaskQueue) *QueueCatalogImportCommand {
	if imports == nil {
		panic("import repository is required")
	}
	if files == nil {
		panic("tenant objects are required")
	}
	if jobQueue == nil {
		panic("job queue is required")
	}
	return &QueueCatalogImportCommand{imports: imports, files: files, jobQueue: jobQueue, now: time.Now}
}

func (cmd *QueueCatalogImportCommand) Execute(ctx context.Context, input QueueCatalogImportInput) (*domain.CatalogImport, error) {
	if !id.IsValidULID(input.ID) {
		return nil, appErrors.New(appErrors.CodeValidation, "import id must be a valid ULID")
	}
	if input.FileKey == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "file_key is required")
	}
	opened, err := cmd.files.Open(ctx, domain.ImportUploadModule, input.FileKey, 0)
	if err != nil {
		return nil, appErrors.Wrap(err, appErrors.CodeValidation, "import file was not found")
	}
	_ = opened.Body.Close()
	imp, err := domain.NewCatalogImport(input.ID, input.FileKey, opened.SizeBytes, input.Requester, cmd.now().UTC())
	if err != nil {
		return nil, appErrors.New(appErrors.CodeValidation, err.Error())
	}
	active, err := cmd.imports.GetActiveImport(ctx)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, appErrors.New(appErrors.CodeConflict, "an import is already in progress")
	}
	if err := cmd.imports.CreateImport(ctx, imp); err != nil {
		return nil, err
	}
	if err := cmd.enqueue(ctx, imp.ID); err != nil {
		if abortErr := cmd.abortQueued(ctx, imp); abortErr != nil {
			return nil, errors.Join(err, abortErr)
		}
		return nil, err
	}
	return &imp, nil
}

func (cmd *QueueCatalogImportCommand) enqueue(ctx context.Context, importID string) error {
	payload, err := contractJobs.MarshalCatalogImportRequested(contractJobs.CatalogImportRequestedJob{ImportID: importID})
	if err != nil {
		return err
	}
	return cmd.jobQueue.Enqueue(ctx, jobs.Job{Type: contractJobs.CatalogImportRequestedType, Payload: payload})
}

func (cmd *QueueCatalogImportCommand) abortQueued(ctx context.Context, imp domain.CatalogImport) error {
	if err := imp.Fail("no se pudo encolar el trabajo", cmd.now().UTC()); err != nil {
		return err
	}
	return cmd.imports.UpdateImport(ctx, imp)
}
