package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	platformstorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
)

type fakeS3Client struct {
	objects map[string][]byte
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	if _, ok := f.objects[*params.Key]; !ok {
		return nil, errors.New("status code: 404")
	}
	return &awss3.HeadObjectOutput{}, nil
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.objects[*params.Key] = body
	return &awss3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	body, ok := f.objects[*params.Key]
	if !ok {
		return nil, errors.New("status code: 404")
	}
	return &awss3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func TestWriteFileIfAbsentUploadsFirstTime(t *testing.T) {
	store := NewObjectStoreWithClient(&fakeS3Client{}, "bucket")

	res, err := store.WriteFileIfAbsent(context.Background(), platformstorage.WriteFileIfAbsentInput{
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

	res, err := store.WriteFileIfAbsent(context.Background(), platformstorage.WriteFileIfAbsentInput{
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

	body, err := store.ReadFile(context.Background(), platformstorage.ReadFileInput{
		Path: "tenant/t/inbox/raw/key",
	})
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if string(body) != "abc" {
		t.Fatalf("unexpected object body: %s", string(body))
	}
}
