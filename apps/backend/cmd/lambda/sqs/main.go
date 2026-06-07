package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	"github.com/bowerbird/internal/platform"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
)

var jobHandler platformJobs.Handler

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	invoicesApp := invoicesModule.NewApplication(
		platformModule.Config,
		platformModule.EventBus,
		platformModule.JobQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
	)

	processorCommand := invoicesJobs.NewInvoiceExtractionRequestedProcessor(
		invoicesApp.Commands.ProcessInvoiceExtractionJob,
	)

	jobHandler = platformJobs.NewHandler(processorCommand)
}

func handle(ctx context.Context, event events.SQSEvent) error {
	return jobHandler.HandleSQSEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
