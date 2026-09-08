package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	"github.com/bowerbird/internal/platform/http/apigateway"
	httphost "github.com/bowerbird/internal/platform/http/host"
)

func main() {
	deps, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	handler, err := httphost.New(deps)
	if err != nil {
		log.Fatalf("failed to wire http api: %v", err)
	}

	lambda.Start(apigateway.ProxyWithContext(handler))
}
