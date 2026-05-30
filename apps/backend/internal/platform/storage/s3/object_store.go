package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type apiClient interface {
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
}

type ObjectStore struct {
	client apiClient
	bucket string
}

var _ platformstorage.FileStore = (*ObjectStore)(nil)

func NewObjectStore(client *awss3.Client, bucket string) *ObjectStore {
	return &ObjectStore{client: client, bucket: bucket}
}

func NewObjectStoreWithClient(client apiClient, bucket string) *ObjectStore {
	return &ObjectStore{client: client, bucket: bucket}
}

func (s *ObjectStore) WriteFileIfAbsent(ctx context.Context, input platformstorage.WriteFileIfAbsentInput) (*platformstorage.WriteFileIfAbsentResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	if len(input.Data) == 0 {
		return nil, fmt.Errorf("data is required")
	}

	_, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.Path),
	})
	if err == nil {
		return &platformstorage.WriteFileIfAbsentResult{Written: true, SizeBytes: int64(len(input.Data))}, nil
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
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(input.Path),
		Body:        bytes.NewReader(input.Data),
		ContentType: aws.String(defaultContentType(input.ContentType)),
		Metadata:    input.Metadata,
	}

	if _, err := s.client.PutObject(ctx, putInput); err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}

	return &platformstorage.WriteFileIfAbsentResult{Written: true, SizeBytes: int64(len(input.Data))}, nil
}

func (s *ObjectStore) ReadFile(ctx context.Context, input platformstorage.ReadFileInput) ([]byte, error) {
	if s.client == nil {
		return nil, fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	res, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.Path),
	})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}

	return body, nil
}

func defaultContentType(contentType string) string {
	if strings.TrimSpace(contentType) == "" {
		return "application/octet-stream"
	}
	return contentType
}
