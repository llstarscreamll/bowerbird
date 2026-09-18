package main

import (
	"context"
	"log"

	"github.com/atta/internal/platform"
	"github.com/atta/internal/platform/http/apigateway"
	httphost "github.com/atta/internal/platform/http/host"
	"github.com/aws/aws-lambda-go/lambda"
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
