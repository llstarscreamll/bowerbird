package jobs

import (
	"github.com/atta/internal/invoices/adapters/jobs/handlers"
	commands "github.com/atta/internal/invoices/application/commands"
)

func NewInvoiceExtractionRequestedProcessor(command *commands.CreateInvoicesFromFilesCommand) *handlers.ProcessInvoiceExtractionFromFiles {
	return handlers.NewProcessInvoiceExtractionFromFiles(command)
}

func NewInvoiceInboxBackfillProcessor(command *commands.BackfillInboxInvoicesCommand) *handlers.ProcessInvoiceInboxBackfill {
	return handlers.NewProcessInvoiceInboxBackfill(command)
}
