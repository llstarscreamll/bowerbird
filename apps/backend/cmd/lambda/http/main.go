package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform/config"
)

var cfg config.Config

func init() {
	var err error
	cfg, err = config.Load(context.Background())
	if err != nil {
		log.Fatalf("failed to load config at boot: %v", err)
	}
}

func handler(_ context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if request.RawPath == "/health" || request.RawPath == "/api/health" {
		body, _ := json.Marshal(map[string]string{"status": "ok"})
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusOK,
			Body:       string(body),
			Headers: map[string]string{
				"Content-Type":                 "application/json",
				"X-Content-Type-Options":       "nosniff",
				"X-Frame-Options":              "DENY",
				"Referrer-Policy":              "strict-origin-when-cross-origin",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNotFound, Body: `{"message":"not found"}`}, nil
}

func main() {
	lambda.Start(handler)
}
