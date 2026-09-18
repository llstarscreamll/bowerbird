package jobs

import (
	"encoding/json"
	"errors"
)

const InvoiceInboxBackfillRequestedType = "InvoiceInboxBackfillRequested"

type InboxBackfillJob struct {
	ID       string `json:"job_id"`
	Cursor   string `json:"cursor,omitempty"`
	QueuedAt string `json:"requested_at"`
}

func (j InboxBackfillJob) Validate() error {
	if j.ID == "" {
		return errors.New("job_id is required")
	}
	return nil
}

func MarshalInboxBackfillRequested(job InboxBackfillJob) ([]byte, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(job)
}

func UnmarshalInboxBackfillRequested(data []byte) (InboxBackfillJob, error) {
	var job InboxBackfillJob
	if err := json.Unmarshal(data, &job); err != nil {
		return InboxBackfillJob{}, err
	}
	if err := job.Validate(); err != nil {
		return InboxBackfillJob{}, err
	}
	return job, nil
}
