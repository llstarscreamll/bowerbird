package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	platformStorage "github.com/bowerbird/internal/platform/storage"
)

type fakeS3Client struct {
	objects map[string][]byte
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *awsS3.HeadObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.HeadObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	if _, ok := f.objects[*params.Key]; !ok {
		return nil, errors.New("status code: 404")
	}
	return &awsS3.HeadObjectOutput{}, nil
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.PutObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.objects[*params.Key] = body
	return &awsS3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.GetObjectOutput, error) {
	body, ok := f.objects[*params.Key]
	if !ok {
		return nil, errors.New("status code: 404")
	}
	return &awsS3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func (f *fakeS3Client) CopyObject(ctx context.Context, params *awsS3.CopyObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.CopyObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	source := *params.CopySource
	_, sourceKey, found := strings.Cut(source, "/")
	if !found {
		return nil, errors.New("invalid copy source")
	}
	body, ok := f.objects[sourceKey]
	if !ok {
		return nil, errors.New("status code: 404")
	}
	f.objects[*params.Key] = body
	return &awsS3.CopyObjectOutput{}, nil
}

func (f *fakeS3Client) DeleteObject(ctx context.Context, params *awsS3.DeleteObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.DeleteObjectOutput, error) {
	if f.objects != nil {
		delete(f.objects, *params.Key)
	}
	return &awsS3.DeleteObjectOutput{}, nil
}

type fakePresignClient struct{}

func (f fakePresignClient) PresignPutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	return &v4.PresignedHTTPRequest{URL: "https://example.test/upload"}, nil
}

func (f fakePresignClient) PresignGetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	return &v4.PresignedHTTPRequest{URL: "https://example.test/download"}, nil
}

func TestWriteFileIfAbsentUploadsFirstTime(t *testing.T) {
	store := NewObjectStoreWithClient(&fakeS3Client{}, "bucket")

	res, err := store.WriteFileIfAbsent(context.Background(), platformStorage.WriteFileIfAbsentInput{
		Path: "tenant/t/inbox/raw/key",
		Data: []byte("abc"),
	})
	if err != nil {
		t.Fatalf("write file if absent failed: %v", err)
	}
	if !res.Written {
		t.Fatal("expected written=true")
	}
}

func TestWriteFileIfAbsentSkipsWhenExists(t *testing.T) {
	client := &fakeS3Client{objects: map[string][]byte{"tenant/t/inbox/raw/key": []byte("abc")}}
	store := NewObjectStoreWithClient(client, "bucket")

	res, err := store.WriteFileIfAbsent(context.Background(), platformStorage.WriteFileIfAbsentInput{
		Path: "tenant/t/inbox/raw/key",
		Data: []byte("abc"),
	})
	if err != nil {
		t.Fatalf("write file if absent failed: %v", err)
	}
	if !res.Written {
		t.Fatal("expected written=true when object already exists")
	}
}

func TestReadFileReadsObjectContent(t *testing.T) {
	client := &fakeS3Client{objects: map[string][]byte{"tenant/t/inbox/raw/key": []byte("abc")}}
	store := NewObjectStoreWithClient(client, "bucket")

	body, err := store.ReadFile(context.Background(), platformStorage.ReadFileInput{
		Path: "tenant/t/inbox/raw/key",
	})
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if string(body) != "abc" {
		t.Fatalf("unexpected object body: %s", string(body))
	}
}

func TestExistsReturnsTrueWhenObjectExists(t *testing.T) {
	client := &fakeS3Client{objects: map[string][]byte{"tenant/t/inbox/raw/key": []byte("abc")}}
	store := NewObjectStoreWithClient(client, "bucket")

	exists, err := store.Exists(context.Background(), platformStorage.ExistsFileInput{Path: "tenant/t/inbox/raw/key"})
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestMoveFileCopiesAndDeletesSource(t *testing.T) {
	client := &fakeS3Client{objects: map[string][]byte{"source": []byte("abc")}}
	store := NewObjectStoreWithClient(client, "bucket")

	err := store.MoveFile(context.Background(), platformStorage.MoveFileInput{SourcePath: "source", DestinationPath: "destination"})
	if err != nil {
		t.Fatalf("move file failed: %v", err)
	}
	if _, ok := client.objects["source"]; ok {
		t.Fatal("expected source key to be deleted")
	}
	if _, ok := client.objects["destination"]; !ok {
		t.Fatal("expected destination key to exist")
	}
}

func TestPresignUploadReturnsURLAndReference(t *testing.T) {
	store := NewObjectStoreWithClients(&fakeS3Client{}, fakePresignClient{}, "bucket")

	result, err := store.PresignUpload(context.Background(), platformStorage.PresignUploadInput{
		Path:        "1-day/t1/uploads/u1/file.bin",
		ContentType: "application/octet-stream",
		ExpiresIn:   10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("presign upload failed: %v", err)
	}
	if result.URL == "" {
		t.Fatal("expected non-empty URL")
	}
	if result.Reference.Key != "1-day/t1/uploads/u1/file.bin" {
		t.Fatalf("unexpected reference key: %s", result.Reference.Key)
	}
}

func TestPresignDownloadReturnsURLAndReference(t *testing.T) {
	store := NewObjectStoreWithClients(&fakeS3Client{}, fakePresignClient{}, "bucket")

	result, err := store.PresignDownload(context.Background(), platformStorage.PresignDownloadInput{
		Path:      "1-day/t1/uploads/u1/file.bin",
		ExpiresIn: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("presign download failed: %v", err)
	}
	if result.URL == "" {
		t.Fatal("expected non-empty URL")
	}
	if result.Method != "GET" {
		t.Fatalf("unexpected method: %s", result.Method)
	}
	if result.Reference.Key != "1-day/t1/uploads/u1/file.bin" {
		t.Fatalf("unexpected reference key: %s", result.Reference.Key)
	}
}
