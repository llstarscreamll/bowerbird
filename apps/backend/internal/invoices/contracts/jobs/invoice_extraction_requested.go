package jobs

import (
	"encoding/json"
	"errors"
)

const (
	InvoiceExtractionRequestedType = "InvoiceExtractionRequested"
)

type File struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type ExtractInvoicesFromFilesJob struct {
	ID         string `json:"job_id"`
	SourceName string `json:"source_name"`
	SourceID   string `json:"source_id"`
	Files      []File `json:"files"`
	QueuedAt   string `json:"requested_at"`
}

func (j ExtractInvoicesFromFilesJob) Validate() error {
	if j.ID == "" {
		return errors.New("job_id is required")
	}
	if j.SourceName == "" {
		return errors.New("source_name is required")
	}
	if j.SourceID == "" {
		return errors.New("source_id is required")
	}
	if len(j.Files) == 0 {
		return errors.New("files is required")
	}

	return nil
}

func MarshalInvoiceExtractionRequested(job ExtractInvoicesFromFilesJob) ([]byte, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(job)
}

func UnmarshalInvoiceExtractionRequested(data []byte) (ExtractInvoicesFromFilesJob, error) {
	var job ExtractInvoicesFromFilesJob
	if err := json.Unmarshal(data, &job); err != nil {
		return ExtractInvoicesFromFilesJob{}, err
	}

	if err := job.Validate(); err != nil {
		return ExtractInvoicesFromFilesJob{}, err
	}

	return job, nil
}
