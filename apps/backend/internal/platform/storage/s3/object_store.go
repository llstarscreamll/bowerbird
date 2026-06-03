package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type apiClient interface {
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	CopyObject(ctx context.Context, params *awss3.CopyObjectInput, optFns ...func(*awss3.Options)) (*awss3.CopyObjectOutput, error)
	DeleteObject(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
}

type ObjectStore struct {
	client         apiClient
	presignClient  presignAPIClient
	bucket         string
	uploadDuration time.Duration
}

type presignAPIClient interface {
	PresignPutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignGetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

var _ platformstorage.FileStore = (*ObjectStore)(nil)

func NewObjectStore(client *awss3.Client, bucket string) *ObjectStore {
	return &ObjectStore{
		client:         client,
		presignClient:  awss3.NewPresignClient(client),
		bucket:         bucket,
		uploadDuration: 15 * time.Minute,
	}
}

func NewObjectStoreWithClient(client apiClient, bucket string) *ObjectStore {
	return &ObjectStore{client: client, bucket: bucket, uploadDuration: 15 * time.Minute}
}

func NewObjectStoreWithClients(client apiClient, presignClient presignAPIClient, bucket string) *ObjectStore {
	return &ObjectStore{client: client, presignClient: presignClient, bucket: bucket, uploadDuration: 15 * time.Minute}
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

	if !isNotFoundError(err) {
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

func (s *ObjectStore) Exists(ctx context.Context, input platformstorage.ExistsFileInput) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return false, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return false, fmt.Errorf("path is required")
	}

	_, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.Path),
	})
	if err == nil {
		return true, nil
	}
	if isNotFoundError(err) {
		return false, nil
	}

	return false, fmt.Errorf("head object: %w", err)
}

func (s *ObjectStore) MoveFile(ctx context.Context, input platformstorage.MoveFileInput) error {
	if s.client == nil {
		return fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.SourcePath) == "" {
		return fmt.Errorf("source path is required")
	}
	if strings.TrimSpace(input.DestinationPath) == "" {
		return fmt.Errorf("destination path is required")
	}
	if input.SourcePath == input.DestinationPath {
		return nil
	}

	copySource := s.bucket + "/" + input.SourcePath
	if _, err := s.client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:            aws.String(s.bucket),
		Key:               aws.String(input.DestinationPath),
		CopySource:        aws.String(copySource),
		MetadataDirective: awss3types.MetadataDirectiveCopy,
	}); err != nil {
		return fmt.Errorf("copy object: %w", err)
	}

	if _, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.SourcePath),
	}); err != nil {
		return fmt.Errorf("delete source object: %w", err)
	}

	return nil
}

func (s *ObjectStore) PresignUpload(ctx context.Context, input platformstorage.PresignUploadInput) (*platformstorage.PresignUploadResult, error) {
	if s.presignClient == nil {
		return nil, fmt.Errorf("s3 presign client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	expiresIn := input.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = s.uploadDuration
	}

	presignedRequest, err := s.presignClient.PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(input.Path),
		ContentType: aws.String(defaultContentType(input.ContentType)),
		Metadata:    input.Metadata,
	}, func(opts *awss3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}

	return &platformstorage.PresignUploadResult{
		URL:    presignedRequest.URL,
		Method: "PUT",
		Headers: map[string]string{
			"Content-Type": defaultContentType(input.ContentType),
		},
		ExpiresAt: time.Now().Add(expiresIn),
		Reference: platformstorage.FileReference{
			Bucket: s.bucket,
			Key:    input.Path,
		},
		UploadPath: input.Path,
	}, nil
}

func (s *ObjectStore) PresignDownload(ctx context.Context, input platformstorage.PresignDownloadInput) (*platformstorage.PresignDownloadResult, error) {
	if s.presignClient == nil {
		return nil, fmt.Errorf("s3 presign client is required")
	}
	if strings.TrimSpace(s.bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	expiresIn := input.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = s.uploadDuration
	}

	presignedRequest, err := s.presignClient.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.Path),
	}, func(opts *awss3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return nil, fmt.Errorf("presign get object: %w", err)
	}

	return &platformstorage.PresignDownloadResult{
		URL:       presignedRequest.URL,
		Method:    "GET",
		ExpiresAt: time.Now().Add(expiresIn),
		Reference: platformstorage.FileReference{
			Bucket: s.bucket,
			Key:    input.Path,
		},
	}, nil
}

func defaultContentType(contentType string) string {
	if strings.TrimSpace(contentType) == "" {
		return "application/octet-stream"
	}
	return contentType
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var notFound *awss3types.NotFound
	if errors.As(err, &notFound) {
		return true
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "not found") ||
		strings.Contains(errText, "status code: 404") ||
		strings.Contains(errText, "statuscode: 404") ||
		strings.Contains(errText, "nosuchkey")
}
