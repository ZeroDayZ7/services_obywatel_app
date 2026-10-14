package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
}

func newMinioClient(endpoint, access, secret string, useSSL bool) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	return client, nil
}

// #region NewS3Storage
func NewS3Storage(internalEndpoint, externalEndpoint, access, secret, bucket string, useSSL bool) (*S3Storage, error) {
	internalClient, err := newMinioClient(internalEndpoint, access, secret, useSSL)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, err := internalClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		err = internalClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	presignEndpoint := strings.TrimSpace(externalEndpoint)
	if presignEndpoint == "" {
		return nil, fmt.Errorf("MINIO_EXTERNAL_ENDPOINT is required for presigned client URLs")
	}
	presignClient, err := newMinioClient(presignEndpoint, access, secret, useSSL)
	if err != nil {
		return nil, err
	}

	return &S3Storage{client: internalClient, presignClient: presignClient, bucket: bucket}, nil
}

// #region Upload
func (s *S3Storage) Upload(ctx context.Context, name string, reader io.Reader, size int64, contentType string) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType, // np. "application/pdf"
	}
	_, err := s.client.PutObject(ctx, s.bucket, name, reader, size, opts)
	return name, err
}

// #region Delete
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// #region Download
func (s *S3Storage) Download(ctx context.Context, key string) ([]byte, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	return io.ReadAll(object)
}

// GetPresignedURL generuje tymczasowy link do bezpiecznego pobrania pliku przez obywatela/urzędnika
// #region GetPresignedURL
func (s *S3Storage) presignClientFor() *minio.Client {
	if s.presignClient != nil {
		return s.presignClient
	}
	return s.client
}

func (s *S3Storage) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presignedURL, err := s.presignClientFor().PresignedGetObject(ctx, s.bucket, key, expires, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (s *S3Storage) GetPresignedUploadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presignedURL, err := s.presignClientFor().PresignedPutObject(ctx, s.bucket, key, expires)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (s *S3Storage) HeadObject(ctx context.Context, key string) (*ObjectMetadata, error) {
	obj, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &ObjectMetadata{
		ContentType:  obj.ContentType,
		Size:         obj.Size,
		ETag:         obj.ETag,
		LastModified: obj.LastModified,
	}, nil
}
