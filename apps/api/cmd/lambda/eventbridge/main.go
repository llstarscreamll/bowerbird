package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/money-path/bowerbird/apps/api/internal/platform/config"
	platformevents "github.com/money-path/bowerbird/apps/api/internal/platform/events"
)

var cfg config.Config
var eventHandler platformevents.EventHandler

func init() {
	var err error
	cfg, err = config.Load(context.Background())
	if err != nil {
		log.Fatalf("failed to load config at boot: %v", err)
	}

	eventHandler = platformevents.NewEventHandler()
}

func handle(ctx context.Context, event events.CloudWatchEvent) error {
	return eventHandler.HandleEventBridgeEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
