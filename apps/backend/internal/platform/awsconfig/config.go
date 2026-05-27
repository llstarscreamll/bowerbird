package awsconfig

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfigv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func Load(ctx context.Context, region, endpointURL, accessKeyID, secretAccessKey string) (aws.Config, error) {
	options := []func(*awsconfigv2.LoadOptions) error{
		awsconfigv2.WithRegion(region),
	}

	if endpointURL != "" {
		options = append(options,
			awsconfigv2.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		)
	}

	return awsconfigv2.LoadDefaultConfig(ctx, options...)
}

func NewSQSClient(awsCfg aws.Config, endpointURL string) *sqs.Client {
	if endpointURL == "" {
		return sqs.NewFromConfig(awsCfg)
	}

	return sqs.NewFromConfig(awsCfg, func(options *sqs.Options) {
		options.BaseEndpoint = &endpointURL
	})
}

func NewS3Client(awsCfg aws.Config, endpointURL string) *s3.Client {
	if endpointURL == "" {
		return s3.NewFromConfig(awsCfg)
	}

	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = &endpointURL
		options.UsePathStyle = true
	})
}

func NewEventBridgeClient(awsCfg aws.Config, endpointURL string) *eventbridge.Client {
	if endpointURL == "" {
		return eventbridge.NewFromConfig(awsCfg)
	}

	return eventbridge.NewFromConfig(awsCfg, func(options *eventbridge.Options) {
		options.BaseEndpoint = &endpointURL
	})
}
