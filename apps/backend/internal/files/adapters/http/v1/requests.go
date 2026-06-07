package v1

import (
	"fmt"
	"strings"
)

type requestUploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Module      string `json:"module"`
}

func (r requestUploadURLRequest) Validate() error {
	if strings.TrimSpace(r.Filename) == "" {
		return fmt.Errorf("filename is required")
	}

	if strings.TrimSpace(r.ContentType) == "" {
		return fmt.Errorf("content_type is required")
	}

	if strings.TrimSpace(r.Module) == "" {
		return fmt.Errorf("module is required")
	}

	return nil
}

type requestDownloadURLRequest struct {
	Key string `json:"key"`
}

func (r requestDownloadURLRequest) Validate() error {
	if strings.TrimSpace(r.Key) == "" {
		return fmt.Errorf("key is required")
	}

	return nil
}
