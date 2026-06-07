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

type InvoiceExtractionRequested struct {
	JobID    string `json:"job_id"`
	Source   string `json:"source"`
	Files    []File `json:"files"`
	QueuedAt string `json:"requested_at"`
}

func (j InvoiceExtractionRequested) Validate() error {
	if j.JobID == "" {
		return errors.New("job_id is required")
	}
	if j.Source == "" {
		return errors.New("source is required")
	}
	if len(j.Files) == 0 {
		return errors.New("files is required")
	}

	return nil
}

func MarshalInvoiceExtractionRequested(job InvoiceExtractionRequested) ([]byte, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(job)
}

func UnmarshalInvoiceExtractionRequested(data []byte) (InvoiceExtractionRequested, error) {
	var job InvoiceExtractionRequested
	if err := json.Unmarshal(data, &job); err != nil {
		return InvoiceExtractionRequested{}, err
	}

	if err := job.Validate(); err != nil {
		return InvoiceExtractionRequested{}, err
	}

	return job, nil
}
