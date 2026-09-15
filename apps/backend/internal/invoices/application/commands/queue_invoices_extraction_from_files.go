package commands

import (
	"context"
	"time"

	"github.com/bowerbird/internal/invoices/application/ports"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
)

type File struct {
	Name     string
	Path     string
	MimeType string
}

type QueueInvoiceExtractionFromFilesInput struct {
	ID    string
	Files []File
}

type QueueInvoiceExtractionFromFilesResult struct {
	JobID            string
	QueuedFilesCount int
}

type QueueInvoiceExtractionFromFilesCommand struct {
	jobQueue  jobs.TaskQueue
	receivers ports.ReceiverDirectory
	now       func() time.Time
	newID     func() string
}

func NewQueueInvoiceExtractionFromFilesCommand(jobQueue jobs.TaskQueue, receivers ports.ReceiverDirectory) *QueueInvoiceExtractionFromFilesCommand {
	if jobQueue == nil {
		panic("job queue is required")
	}
	if receivers == nil {
		panic("receiver directory is required")
	}

	return &QueueInvoiceExtractionFromFilesCommand{
		jobQueue:  jobQueue,
		receivers: receivers,
		now:       time.Now,
		newID:     id.NewULID,
	}
}

func (cmd *QueueInvoiceExtractionFromFilesCommand) Execute(ctx context.Context, input QueueInvoiceExtractionFromFilesInput) (*QueueInvoiceExtractionFromFilesResult, error) {
	hasReceiver, err := cmd.receivers.HasAny(ctx)
	if err != nil {
		return nil, err
	}
	if !hasReceiver {
		return nil, appErrors.New(appErrors.CodeConflict, "a legal entity is required before extracting invoices")
	}
	files := make([]contractJobs.File, 0, len(input.Files))
	for _, file := range input.Files {
		files = append(files, contractJobs.File{
			Path:     file.Path,
			Filename: file.Name,
			MimeType: file.MimeType,
		})
	}

	jobID := input.ID
	if jobID == "" {
		jobID = cmd.newID()
	}

	job := contractJobs.ExtractInvoicesFromFilesJob{
		ID:         jobID,
		SourceName: "files-uploaded-by-user",
		SourceID:   jobID,
		Files:      files,
		QueuedAt:   cmd.now().UTC().Format(time.RFC3339Nano),
	}

	payload, err := contractJobs.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return nil, err
	}

	err = cmd.jobQueue.Enqueue(ctx, jobs.Job{
		Type:    contractJobs.InvoiceExtractionRequestedType,
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	return &QueueInvoiceExtractionFromFilesResult{
		JobID:            job.ID,
		QueuedFilesCount: len(files),
	}, nil
}
