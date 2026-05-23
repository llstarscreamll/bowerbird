package awsconfig

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfigv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
