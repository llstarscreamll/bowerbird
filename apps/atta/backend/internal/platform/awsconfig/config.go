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

	cfg, err := awsconfigv2.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return aws.Config{}, err
	}

	// MinIO and other S3-compatible stores reject default SDK CRC32 checksum trailers (SignatureDoesNotMatch on PutObject).
	if endpointURL != "" {
		cfg.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		cfg.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	}

	return cfg, nil
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
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
}

func NewS3PresignClient(awsCfg aws.Config, endpointURL string) *s3.PresignClient {
	if endpointURL == "" {
		return s3.NewPresignClient(NewS3Client(awsCfg, ""))
	}
	return s3.NewPresignClient(NewS3Client(awsCfg, endpointURL))
}

func NewEventBridgeClient(awsCfg aws.Config, endpointURL string) *eventbridge.Client {
	if endpointURL == "" {
		return eventbridge.NewFromConfig(awsCfg)
	}

	return eventbridge.NewFromConfig(awsCfg, func(options *eventbridge.Options) {
		options.BaseEndpoint = &endpointURL
	})
}
