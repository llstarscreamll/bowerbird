package s3

import (
	"context"
	"errors"
	"testing"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3Client struct {
	objects map[string]struct{}
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string]struct{}{}
	}
	if _, ok := f.objects[*params.Key]; !ok {
		return nil, errors.New("status code: 404")
	}
	return &awss3.HeadObjectOutput{}, nil
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	if f.objects == nil {
		f.objects = map[string]struct{}{}
	}
	f.objects[*params.Key] = struct{}{}
	return &awss3.PutObjectOutput{}, nil
}

func TestPutObjectIfAbsentUploadsFirstTime(t *testing.T) {
	store := NewObjectStoreWithClient(&fakeS3Client{})

	res, err := store.PutObjectIfAbsent(context.Background(), PutObjectIfAbsentInput{
		Bucket: "bucket",
		Key:    "tenant/t/inbox/raw/key",
		Data:   []byte("abc"),
	})
	if err != nil {
		t.Fatalf("put object if absent failed: %v", err)
	}
	if !res.Uploaded {
		t.Fatal("expected uploaded=true")
	}
}

func TestPutObjectIfAbsentSkipsWhenExists(t *testing.T) {
	client := &fakeS3Client{objects: map[string]struct{}{"tenant/t/inbox/raw/key": {}}}
	store := NewObjectStoreWithClient(client)

	res, err := store.PutObjectIfAbsent(context.Background(), PutObjectIfAbsentInput{
		Bucket: "bucket",
		Key:    "tenant/t/inbox/raw/key",
		Data:   []byte("abc"),
	})
	if err != nil {
		t.Fatalf("put object if absent failed: %v", err)
	}
	if !res.Uploaded {
		t.Fatal("expected uploaded=true when object already exists")
	}
}
