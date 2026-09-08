package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
	"github.com/bowerbird/internal/platform/outbox/relay"
)

var runner *relay.MultiTenantRelay

func init() {
	deps, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	transport, _, err := platformMessaging.NewBrokerTransport(deps)
	if err != nil {
		log.Fatalf("failed to build broker transport: %v", err)
	}

	lister := relay.NewControlPlaneTenantLister(deps.ControlDB)
	runner = relay.NewMultiTenantRelay(deps.TenantRegistry, lister, transport, relay.Config{BatchSize: 50, PerTenantCap: 10})
}

func handle(ctx context.Context, _ events.CloudWatchEvent) error {
	if err := runner.RunOnce(ctx); err != nil {
		log.Printf("outbox relay error: %v", err)
		return err
	}
	return nil
}

func main() {
	lambda.Start(handle)
}
