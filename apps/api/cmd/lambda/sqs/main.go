package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/money-path/turno/apps/api/internal/handlers"
)

func handle(ctx context.Context, event events.SQSEvent) error {
	return handlers.NewEventHandler().HandleSQSEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
