package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type apiClient interface {
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
}

type ObjectStore struct {
	client apiClient
}

type PutObjectIfAbsentInput struct {
	Bucket      string
	Key         string
	Data        []byte
	ContentType string
	Metadata    map[string]string
}

type PutObjectIfAbsentResult struct {
	Uploaded  bool
	SizeBytes int64
}

func NewObjectStore(client *awss3.Client) *ObjectStore {
	return &ObjectStore{client: client}
}

func NewObjectStoreWithClient(client apiClient) *ObjectStore {
	return &ObjectStore{client: client}
}

func (s *ObjectStore) PutObjectIfAbsent(ctx context.Context, input PutObjectIfAbsentInput) (*PutObjectIfAbsentResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(input.Bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Key) == "" {
		return nil, fmt.Errorf("key is required")
	}
	if len(input.Data) == 0 {
		return nil, fmt.Errorf("data is required")
	}

	_, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})
	if err == nil {
		return &PutObjectIfAbsentResult{Uploaded: true, SizeBytes: int64(len(input.Data))}, nil
	}

	var notFound *awss3types.NotFound
	if !errors.As(err, &notFound) &&
		!strings.Contains(strings.ToLower(err.Error()), "not found") &&
		!strings.Contains(strings.ToLower(err.Error()), "status code: 404") &&
		!strings.Contains(strings.ToLower(err.Error()), "statuscode: 404") &&
		!strings.Contains(strings.ToLower(err.Error()), "nosuchkey") {
		return nil, fmt.Errorf("head object: %w", err)
	}

	putInput := &awss3.PutObjectInput{
		Bucket:      aws.String(input.Bucket),
		Key:         aws.String(input.Key),
		Body:        bytes.NewReader(input.Data),
		ContentType: aws.String(defaultContentType(input.ContentType)),
		Metadata:    input.Metadata,
	}

	if _, err := s.client.PutObject(ctx, putInput); err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}

	return &PutObjectIfAbsentResult{Uploaded: true, SizeBytes: int64(len(input.Data))}, nil
}

func defaultContentType(contentType string) string {
	if strings.TrimSpace(contentType) == "" {
		return "application/octet-stream"
	}
	return contentType
}
