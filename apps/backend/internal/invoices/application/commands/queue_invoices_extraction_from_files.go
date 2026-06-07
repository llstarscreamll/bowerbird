package commands

import (
	"context"
	"time"

	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/jobs"
)

type File struct {
	Name     string
	Path     string
	MimeType string
}

type QueueInvoiceExtractionFromFilesInput struct {
	Files []File
}

type QueueInvoiceExtractionFromFilesResult struct {
	JobID            string
	QueuedFilesCount int
}

type QueueInvoiceExtractionFromFilesCommand struct {
	jobQueue jobs.Queue
	now      func() time.Time
	newID    func() string
}

func NewQueueInvoiceExtractionFromFilesCommand(jobQueue jobs.Queue) *QueueInvoiceExtractionFromFilesCommand {
	if jobQueue == nil {
		panic("job queue is required")
	}

	return &QueueInvoiceExtractionFromFilesCommand{
		jobQueue: jobQueue,
		now:      time.Now,
		newID:    id.NewULID,
	}
}

func (cmd *QueueInvoiceExtractionFromFilesCommand) Execute(ctx context.Context, input QueueInvoiceExtractionFromFilesInput) (*QueueInvoiceExtractionFromFilesResult, error) {
	files := make([]contractJobs.File, 0, len(input.Files))
	for _, file := range input.Files {
		files = append(files, contractJobs.File{
			Path:     file.Path,
			Filename: file.Name,
			MimeType: file.MimeType,
		})
	}

	jobID := cmd.newID()
	job := contractJobs.InvoiceExtractionRequested{
		JobID:    jobID,
		Source:   "files-uploaded-by-user",
		Files:    files,
		QueuedAt: cmd.now().UTC().Format(time.RFC3339Nano),
	}

	payload, err := contractJobs.MarshalInvoiceExtractionRequested(job)
	if err != nil {
		return nil, err
	}

	err = cmd.jobQueue.Dispatch(ctx, jobs.Job{
		Type:    contractJobs.InvoiceExtractionRequestedType,
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}

	return &QueueInvoiceExtractionFromFilesResult{
		JobID:            jobID,
		QueuedFilesCount: len(files),
	}, nil
}
