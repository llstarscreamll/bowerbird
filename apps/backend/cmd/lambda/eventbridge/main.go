package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
)

var eventHandler platformMessaging.Handlers

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}
	eventHandler = platformMessaging.WireMessagingHandlers(platformModule)
}

func handle(ctx context.Context, event events.CloudWatchEvent) error {
	return eventHandler.Events.HandleEventBridgeEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
