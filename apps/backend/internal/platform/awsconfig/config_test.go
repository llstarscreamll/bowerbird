package awsconfig

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestNewS3PresignClientUsesCustomEndpoint(t *testing.T) {
	awsCfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
	presignClient := NewS3PresignClient(awsCfg, "https://media.bowerbird.dev")

	request, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String("bowerbird-local-bucket"),
		Key:    aws.String("tenant/t1/file.pdf"),
	})
	if err != nil {
		t.Fatalf("presign get object failed: %v", err)
	}

	parsed, err := url.Parse(request.URL)
	if err != nil {
		t.Fatalf("parse presigned URL failed: %v", err)
	}

	if parsed.Host != "media.bowerbird.dev" {
		t.Fatalf("expected custom host media.bowerbird.dev, got %s", parsed.Host)
	}

	if !strings.HasPrefix(parsed.Path, "/bowerbird-local-bucket/") {
		t.Fatalf("expected path-style URL with bucket prefix, got %s", parsed.Path)
	}
}
