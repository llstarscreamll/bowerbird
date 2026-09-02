package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
)

var jobHandler platformMessaging.Handlers

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}
	jobHandler = platformMessaging.WireMessagingHandlers(platformModule)
}

func handle(ctx context.Context, event events.SQSEvent) error {
	return jobHandler.Jobs.HandleSQSEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
