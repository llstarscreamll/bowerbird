package jobs

import (
	"encoding/json"
	"errors"
)

const CatalogImportRequestedType = "CatalogImportRequested"

type CatalogImportRequestedJob struct {
	ImportID string `json:"import_id"`
}

func (j CatalogImportRequestedJob) Validate() error {
	if j.ImportID == "" {
		return errors.New("import_id is required")
	}
	return nil
}

func MarshalCatalogImportRequested(job CatalogImportRequestedJob) ([]byte, error) {
	if err := job.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(job)
}

func UnmarshalCatalogImportRequested(data []byte) (CatalogImportRequestedJob, error) {
	var job CatalogImportRequestedJob
	if err := json.Unmarshal(data, &job); err != nil {
		return CatalogImportRequestedJob{}, err
	}
	if err := job.Validate(); err != nil {
		return CatalogImportRequestedJob{}, err
	}
	return job, nil
}
