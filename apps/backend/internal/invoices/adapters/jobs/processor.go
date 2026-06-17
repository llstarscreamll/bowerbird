package jobs

import (
	"github.com/bowerbird/internal/invoices/adapters/jobs/handlers"
	commands "github.com/bowerbird/internal/invoices/application/commands"
)

func NewInvoiceExtractionRequestedProcessor(command *commands.CreateInvoicesFromFilesCommand) *handlers.ProcessInvoiceExtractionFromFiles {
	return handlers.NewProcessInvoiceExtractionFromFiles(command)
}
