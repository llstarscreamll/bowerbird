package v1

import "github.com/bowerbird/internal/invoices/application/commands"

type jsonApiResponse[T any] struct {
	Data jsonApiDocument[T] `json:"data"`
}

type jsonApiDocument[T any] struct {
	Type       string `json:"type,omitempty"`
	ID         string `json:"id,omitempty"`
	Attributes T      `json:"attributes,omitempty"`
}

type queueInvoiceExtractionResponse struct {
	JobID            string `json:"job_id"`
	Status           string `json:"status"`
	QueuedFilesCount int    `json:"queued_files_count"`
}

func newQueueInvoiceExtractionResponse(result *commands.QueueInvoiceExtractionFromFilesResult) jsonApiResponse[queueInvoiceExtractionResponse] {
	return jsonApiResponse[queueInvoiceExtractionResponse]{
		Data: jsonApiDocument[queueInvoiceExtractionResponse]{
			Type: "queue-invoice-extraction",
			ID:   result.JobID,
			Attributes: queueInvoiceExtractionResponse{
				JobID:            result.JobID,
				Status:           "queued",
				QueuedFilesCount: result.QueuedFilesCount,
			},
		},
	}
}
