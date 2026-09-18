package jobs

import (
	"encoding/json"
	"errors"
)

const InboxSyncAccountType = "InboxSyncAccount"

type InboxSyncAccountJob struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	Provider  string `json:"provider"`
}

func (j InboxSyncAccountJob) Validate() error {
	if j.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if j.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}

func MarshalInboxSyncAccount(job InboxSyncAccountJob) ([]byte, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(job)
}

func UnmarshalInboxSyncAccount(data []byte) (InboxSyncAccountJob, error) {
	var job InboxSyncAccountJob
	if err := json.Unmarshal(data, &job); err != nil {
		return InboxSyncAccountJob{}, err
	}
	if err := job.Validate(); err != nil {
		return InboxSyncAccountJob{}, err
	}
	return job, nil
}
