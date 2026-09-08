package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/outbox/relay"
	awsbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/aws"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context) error {
	deps, err := platform.NewModule(ctx)
	if err != nil {
		return err
	}
	defer deps.ControlDB.Close()
	defer deps.TenantRegistry.CloseAll()

	cfg := deps.Config
	transport := awsbroker.NewTransport(
		awsConfig.NewEventBridgeClient(deps.AWSConfig, cfg.AWSEndpointURL),
		awsConfig.NewSQSClient(deps.AWSConfig, cfg.AWSEndpointURL),
		cfg.EventBusName,
		cfg.SQSQueueURL,
		cfg.MessagingAttestationSecret,
	)

	lister := relay.NewControlPlaneTenantLister(deps.ControlDB)
	multi := relay.NewMultiTenantRelay(deps.TenantRegistry, lister, transport, relay.Config{BatchSize: 50, PerTenantCap: 10})
	if err := multi.RunOnce(ctx); err != nil {
		log.Printf("outbox relay error: %v", err)
		return err
	}
	log.Printf("outbox relay completed (target=%s, multi-tenant)", config.DeploymentTargetAWS)
	return nil
}
